package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/buckets"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{}) error
}

type service struct {
	baseLog *logrus.Entry
	env     env.Interface

	dbGroup schedulerDBs

	m              metrics.Emitter
	mu             sync.RWMutex
	cond           *sync.Cond
	stopping       *atomic.Bool
	workers        *atomic.Int32
	workerRoutines sync.WaitGroup

	b        buckets.BucketWorker[*api.MaintenanceScheduleDocument]
	subs     changefeed.SubscriptionsCache
	clusters *openShiftClusterCache

	changefeedBatchSize int
	changefeedInterval  time.Duration

	lastChangefeed atomic.Value //time.Time
	startTime      time.Time

	pollTime    time.Duration
	now         func() time.Time
	workerDelay func() time.Duration

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask

	serveHealthz bool
}

var _ Runnable = (*service)(nil)

type schedulerDBs interface {
	database.DatabaseGroupWithOpenShiftClusters
	database.DatabaseGroupWithSubscriptions
	database.DatabaseGroupWithMaintenanceManifests
	database.DatabaseGroupWithMaintenanceSchedules
}

func NewService(env env.Interface, log *logrus.Entry, dbg schedulerDBs, m metrics.Emitter, ownedBuckets []int) *service {
	s := &service{
		env:     env,
		baseLog: log,

		dbGroup: dbg,

		m:        m,
		stopping: &atomic.Bool{},
		workers:  &atomic.Int32{},

		startTime:   time.Now(),
		workerDelay: func() time.Duration { return time.Duration(rand.Intn(60)) * time.Second },
		now:         time.Now,
		pollTime:    time.Minute,

		changefeedBatchSize: 50,
		changefeedInterval:  10 * time.Second,

		subs: changefeed.NewSubscriptionsChangefeedCache(false),

		serveHealthz: true,
	}

	s.cond = sync.NewCond(&s.mu)
	s.b = buckets.NewBucketWorker[*api.MaintenanceScheduleDocument](log, s.spawnWorker, &s.mu)
	s.b.SetBuckets(ownedBuckets)

	return s
}

func (s *service) SetMaintenanceTasks(tasks map[api.MIMOTaskID]tasks.MaintenanceTask) {
	s.tasks = tasks
}

func (s *service) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) error {
	defer recover.Panic(s.baseLog)

	// Only enable the healthz endpoint if configured (disabled in unit tests)
	if s.serveHealthz {
		c := &healthz.Handler{
			Checks: map[string]healthz.Checker{
				"ready": func(h *http.Request) error {
					if !s.checkReady() {
						return errors.New("not ready")
					}
					return nil
				},
			},
		}

		m := http.NewServeMux()
		m.Handle("/healthz", http.StripPrefix("/healthz", c))
		// Handle healthz subpaths
		m.Handle("/healthz/", http.StripPrefix("/healthz", c))

		h := &http.Server{
			Handler:     m,
			ErrorLog:    log.New(s.baseLog.Writer(), "", 0),
			BaseContext: func(net.Listener) context.Context { return ctx },
		}

		listener, err := s.env.Listen()
		if err != nil {
			return err
		}

		go func() {
			err := h.Serve(listener)
			if err != http.ErrServerClosed {
				s.baseLog.Error(err)
			}
		}()
	}

	dbOpenShiftClusters, err := s.dbGroup.OpenShiftClusters()
	if err != nil {
		return err
	}

	dbSubscriptions, err := s.dbGroup.Subscriptions()
	if err != nil {
		return err
	}

	// start subscription changefeed
	var subChangefeed changefeed.Changefeed[*api.SubscriptionDocuments] = dbSubscriptions.ChangeFeed()
	go changefeed.NewChangefeed(
		ctx, s.baseLog.WithField("component", "changefeed"), subChangefeed,
		s.changefeedInterval,
		s.changefeedBatchSize, s.subs, stop,
	)

	// start cluster changefeed
	var clusterChangefeed changefeed.Changefeed[*api.OpenShiftClusterDocuments] = dbOpenShiftClusters.ChangeFeed()
	go changefeed.NewChangefeed(
		ctx, s.baseLog.WithField("component", "changefeed"), clusterChangefeed,
		s.changefeedInterval,
		s.changefeedBatchSize, s.clusters, stop,
	)

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	if stop != nil {
		go func() {
			defer recover.Panic(s.baseLog)

			<-stop
			s.baseLog.Print("stopping")
			s.stopping.Store(true)
			s.cond.Signal()
		}()
	}
	go heartbeat.EmitHeartbeat(s.baseLog, s.m, "actuator.heartbeat", nil, s.checkReady)

	lastGotDocs := make(map[string]*api.MaintenanceScheduleDocument)
	for !s.stopping.Load() {
		old, err := s.poll(ctx, lastGotDocs)
		if err != nil {
			s.baseLog.Error(err)
		} else {
			lastGotDocs = old
		}

		<-t.C
	}

	if !s.env.FeatureIsSet(env.FeatureDisableReadinessDelay) {
		s.waitForWorkerCompletion()
	}
	s.baseLog.Print("exiting")
	close(done)
	return nil
}

// Temporary method of updating without the changefeed -- the reason why is
// complicated
func (s *service) poll(ctx context.Context, oldDocs map[string]*api.MaintenanceScheduleDocument) (map[string]*api.MaintenanceScheduleDocument, error) {
	dbMaintenanceSchedules, err := s.dbGroup.MaintenanceSchedules()
	if err != nil {
		return nil, err
	}

	// Fetch all of the cluster UUIDs
	i, err := dbMaintenanceSchedules.GetValid(ctx, "")
	if err != nil {
		return nil, err
	}

	docs := make([]*api.MaintenanceScheduleDocument, 0)

	for {
		d, err := i.Next(ctx, -1)
		if err != nil {
			return nil, err
		}
		if d == nil {
			break
		}

		docs = append(docs, d.MaintenanceScheduleDocuments...)
	}

	s.baseLog.Debugf("fetched %d schedule documents from CosmosDB", len(docs))

	docMap := make(map[string]*api.MaintenanceScheduleDocument)
	for _, d := range docs {
		docMap[strings.ToLower(d.ID)] = d
	}

	// remove docs that don't exist in the new set (removed clusters)
	for oldCluster := range oldDocs {
		_, ok := docMap[strings.ToLower(oldCluster)]
		if !ok {
			s.b.DeleteDoc(oldDocs[oldCluster])
			s.baseLog.Debugf("removed %s from buckets", oldCluster)
		}
	}

	s.baseLog.Debugf("updating %d schedules", len(docMap))

	for _, cluster := range docMap {
		s.b.UpsertDoc(cluster)
	}

	// Store when we last fetched the clusters
	s.lastChangefeed.Store(s.now())

	return docMap, nil
}

func (s *service) waitForWorkerCompletion() {
	s.mu.Lock()
	for s.workers.Load() > 0 {
		s.cond.Wait()
	}
	s.mu.Unlock()
}

func (s *service) checkReady() bool {
	lastChangefeedTime, ok := s.lastChangefeed.Load().(time.Time)
	if !ok {
		return false
	}

	lastClusterChangefeed, ok := s.clusters.GetLastProcessed()
	if !ok {
		return false
	}

	lastSubsChangefeed, ok := s.subs.GetLastProcessed()
	if !ok {
		return false
	}

	if s.env.IsLocalDevelopmentMode() {
		return (time.Since(lastChangefeedTime) < time.Minute && // did we update our changefeeds recently?
			time.Since(lastClusterChangefeed) < time.Minute &&
			time.Since(lastSubsChangefeed) < time.Minute)
	} else {
		return (time.Since(lastChangefeedTime) < time.Minute) && // did we update our list of clusters recently?
			(time.Since(s.startTime) > 2*time.Minute) // are we running for at least 2 minutes?
	}
}

func (s *service) spawnWorker(stop <-chan struct{}, id string) {
	s.workerRoutines.Add(1)
	go s.worker(stop, id)
}

func (s *service) worker(stop <-chan struct{}, id string) {
	defer recover.Panic(s.baseLog)
	defer s.workerRoutines.Done()

	delay := s.workerDelay()
	log := s.baseLog.WithFields(logrus.Fields{"scheduleID": id})
	log.Debugf("starting worker for %s in %s...", id, delay.String())

	// Wait for a randomised delay before starting
	time.Sleep(delay)

	getDoc := func() (*api.MaintenanceScheduleDocument, bool) { return s.b.Doc(id) }

	a, err := NewSchedulerForSchedule(context.Background(), s.env, log, getDoc, s.dbGroup, s.now)
	if err != nil {
		log.Error(err)
		return
	}

	// load in valid tasks
	a.AddMaintenanceTasks(s.tasks)

	t := time.NewTicker(s.pollTime)
	defer func() {
		log.Debugf("stopping worker for %s...", id)
		t.Stop()
	}()

out:
	for !s.stopping.Load() {
		func() {
			s.workers.Add(1)
			s.m.EmitGauge("mimo.scheduler.workers.active.count", int64(s.workers.Load()), nil)

			defer func() {
				s.workers.Add(-1)
				s.m.EmitGauge("mimo.scheduler.workers.active.count", int64(s.workers.Load()), nil)
			}()

			_, err := a.Process(context.Background())
			if err != nil {
				log.Error(err)
			}
		}()

		select {
		case <-t.C:
		case <-stop:
			break out
		}
	}
}
