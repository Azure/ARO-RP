package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"iter"
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
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/buckets"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{}) error
}

var (
	defaultWorkerMaxStartupDelay       = 60 * time.Second
	defaultServiceInterval             = 180 * time.Second
	defaultReadinessDelay              = 2 * time.Minute
	defaultSchedulePollInterval        = 30 * time.Second
	defaultSchedulePollReadiness       = 90 * time.Second
	defaultChangefeedInteval           = 10 * time.Second
	defaultChangefeedReadinessInterval = time.Minute
)

type service struct {
	baseLog *logrus.Entry
	env     env.Interface

	dbGroup schedulerDBs

	m              metrics.Emitter
	mu             sync.RWMutex
	stopping       *atomic.Bool
	workers        *atomic.Int32
	workerRoutines sync.WaitGroup
	newScheduler   newSchedulerFunc

	buckets  atomic.Value // []int
	b        buckets.BucketWorker[*api.MaintenanceScheduleDocument]
	subs     changefeed.SubscriptionsCache
	clusters *openShiftClusterCache

	bucketCount         int
	changefeedBatchSize int

	lastScheduleUpdate atomic.Value // time.Time
	lastBucketUpdate   atomic.Value // time.Time
	startTime          time.Time

	workerMaxStartupDelay     time.Duration // Maximum interval before a worker starts
	interval                  time.Duration // Interval between service runs
	schedulePollInterval      time.Duration // Interval between updates to Schedules
	readyIfSchedulePollWithin time.Duration // Time that the Schedules should have been updated within to be ready
	changefeedInterval        time.Duration // Interval between changefeed runs (updates to cluster docs + subscriptions)
	readyIfChangefeedWithin   time.Duration // Time that the changefeed should have been changed within to be healthy
	readinessDelay            time.Duration // Minimal time until the service will allow itself to be marked ready

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask

	serveHealthz  bool
	emitHeartbeat bool
}

var _ Runnable = (*service)(nil)

type schedulerDBs interface {
	database.DatabaseGroupWithOpenShiftClusters
	database.DatabaseGroupWithSubscriptions
	database.DatabaseGroupWithMaintenanceManifests
	database.DatabaseGroupWithMaintenanceSchedules
	database.DatabaseGroupWithPoolWorkers
}

func NewService(env env.Interface, log *logrus.Entry, dbg schedulerDBs, m metrics.Emitter) *service {
	s := &service{
		env:     env,
		baseLog: log,

		dbGroup: dbg,

		m:           m,
		stopping:    &atomic.Bool{},
		workers:     &atomic.Int32{},
		bucketCount: bucket.Buckets,

		startTime:             env.Now(),
		workerMaxStartupDelay: defaultWorkerMaxStartupDelay,
		newScheduler:          NewSchedulerForSchedule,

		changefeedBatchSize:       50,
		changefeedInterval:        defaultChangefeedInteval,
		interval:                  defaultServiceInterval,
		readyIfChangefeedWithin:   defaultChangefeedReadinessInterval,
		readinessDelay:            defaultReadinessDelay,
		schedulePollInterval:      defaultSchedulePollInterval,
		readyIfSchedulePollWithin: defaultSchedulePollReadiness,

		subs: changefeed.NewSubscriptionsChangefeedCache(m, false),

		serveHealthz:  true,
		emitHeartbeat: true,
	}

	s.clusters = newOpenShiftClusterCache(log, m, s.subs)
	s.b = buckets.NewBucketWorker[*api.MaintenanceScheduleDocument](log, s.spawnWorker, &s.mu)
	return s
}

func (s *service) SetMaintenanceTasks(tasks map[api.MIMOTaskID]tasks.MaintenanceTask) {
	s.tasks = tasks
}

func (s *service) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) error {
	defer recover.Panic(s.baseLog)

	dbPoolWorkers, err := s.dbGroup.PoolWorkers()
	if err != nil {
		return err
	}

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

	if stop != nil {
		go func() {
			defer recover.Panic(s.baseLog)

			<-stop
			s.baseLog.Print("stopping")
			s.stopping.Store(true)
		}()
	}

	err = s.startChangefeeds(ctx, stop)
	if err != nil {
		return err
	}

	if s.emitHeartbeat {
		go heartbeat.EmitHeartbeat(s.baseLog, s.m, "scheduler.heartbeat", stop, s.checkReady)
	}
	// Start the bucket worker update loop which will coordinate buckets between
	// the MIMO instances
	go buckets.StartBucketWorkerLoop(
		ctx, s.baseLog, api.PoolWorkerTypeMIMOScheduler,
		s.bucketCount, s.changefeedInterval, dbPoolWorkers, func(i []int) {
			if len(i) == 0 {
				s.baseLog.Error("got an allocation of 0 buckets, ignoring")
				return
			}
			s.buckets.Store(i)
			s.lastBucketUpdate.Store(s.env.Now())
		}, stop,
	)

	t := time.NewTicker(s.schedulePollInterval)
	defer t.Stop()

	lastGotDocs := make(map[string]*api.MaintenanceScheduleDocument)
	for !s.stopping.Load() {
		old, err := s.poll(ctx, lastGotDocs)
		if err != nil {
			s.baseLog.Error(err)
		} else {
			lastGotDocs = old
		}

		select {
		case <-t.C:
		case <-ctx.Done():
			s.baseLog.Warn("context closed, stopping")
			s.stopping.Store(true)
		}
	}

	// If we're here, we're exiting
	s.baseLog.Print("exiting, waiting for all workers to finish")
	s.workerRoutines.Wait()
	close(done)
	return nil
}

func (s *service) startChangefeeds(ctx context.Context, stop <-chan struct{}) error {
	dbOpenShiftClusters, err := s.dbGroup.OpenShiftClusters()
	if err != nil {
		return err
	}

	dbSubscriptions, err := s.dbGroup.Subscriptions()
	if err != nil {
		return err
	}

	// start subscription changefeed
	go changefeed.RunChangefeed(
		ctx, s.baseLog.WithField("component", "changefeed"), dbSubscriptions.ChangeFeed(),
		s.changefeedInterval,
		s.changefeedBatchSize, s.subs, stop,
	)

	// start cluster changefeed
	go changefeed.RunChangefeed(
		ctx, s.baseLog.WithField("component", "changefeed"), dbOpenShiftClusters.ChangeFeed(),
		s.changefeedInterval,
		s.changefeedBatchSize, s.clusters, stop,
	)

	return nil
}

// Temporary method of updating without the changefeed -- the reason why is
// complicated
func (s *service) poll(ctx context.Context, oldDocs map[string]*api.MaintenanceScheduleDocument) (map[string]*api.MaintenanceScheduleDocument, error) {
	dbMaintenanceSchedules, err := s.dbGroup.MaintenanceSchedules()
	if err != nil {
		return nil, err
	}

	// Fetch all of the valid schedules
	i, err := dbMaintenanceSchedules.GetValid(ctx, "")
	if err != nil {
		return nil, err
	}

	docs := make([]*api.MaintenanceScheduleDocument, 0)

	for {
		d, err := i.Next(ctx, s.changefeedBatchSize)
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

	// Acquire lock for when we're mutating the changefeed cache
	s.mu.Lock()
	defer s.mu.Unlock()

	// remove docs that don't exist in the new set (removed schedules)
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

	// Store when we last fetched the schedules
	s.lastScheduleUpdate.Store(s.env.Now())

	// Emit a metric containing the size of our cache
	s.m.EmitGauge("changefeed.caches.size", int64(s.b.Size()), map[string]string{
		"name": "MaintenanceScheduleDocument",
	})

	return docMap, nil
}

func (s *service) checkReady() bool {
	now := s.env.Now()

	lastBucketUpdate, ok := s.lastBucketUpdate.Load().(time.Time)
	if !ok {
		return false
	}

	lastScheduleUpdate, ok := s.lastScheduleUpdate.Load().(time.Time)
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

	return (now.Sub(lastScheduleUpdate) < s.readyIfSchedulePollWithin && // did we update our changefeeds recently?
		now.Sub(lastClusterChangefeed) < s.readyIfChangefeedWithin &&
		now.Sub(lastSubsChangefeed) < s.readyIfChangefeedWithin &&
		now.Sub(lastBucketUpdate) < s.readyIfChangefeedWithin &&
		now.Sub(s.startTime) > s.readinessDelay) // are we running for at least (the default) 2 minutes?
}

func (s *service) spawnWorker(stop <-chan struct{}, id string) {
	s.workerRoutines.Go(func() {
		s.worker(stop, id)
	})
}

func (s *service) worker(stop <-chan struct{}, id string) {
	defer recover.Panic(s.baseLog)

	delay := s.workerMaxStartupDelay * time.Duration(rand.Float32())
	log := s.baseLog.WithFields(logrus.Fields{"scheduleID": id})
	log.Debugf("starting worker for %s in %s...", id, delay.String())

	// Wait for a randomised delay before starting
	time.Sleep(delay)

	getDoc := func() (*api.MaintenanceScheduleDocument, bool) { return s.b.Doc(id) }
	getClusters := func() iter.Seq2[string, selectorData] {
		return func(yield func(string, selectorData) bool) {
			_ownedBuckets, ok := s.buckets.Load().([]int)
			if !ok {
				// no owned buckets yet
				return
			}

			ownedBuckets := make(map[string]struct{})
			for _, i := range _ownedBuckets {
				ownedBuckets[fmt.Sprintf("%d", i)] = struct{}{}
			}

			// Only give clusters belonging to buckets we currently have owned
			for cl, d := range s.clusters.GetClusters() {
				bucket, ok := d.GetString(string(SelectorDataBucketID))
				if !ok {
					continue
				}

				_, ownedBucket := ownedBuckets[bucket]
				if ownedBucket {
					if !yield(cl, d) {
						return
					}
				}
			}
		}
	}

	a, err := s.newScheduler(s.env, log, s.m, getDoc, getClusters, s.dbGroup)
	if err != nil {
		log.Error(err)
		return
	}

	// load in valid tasks
	a.AddMaintenanceTasks(s.tasks)

	t := time.NewTicker(s.interval)
	defer func() {
		log.Debugf("stopping worker for %s...", id)
		t.Stop()
	}()

out:
	for {
		select {
		case <-t.C:
			if s.stopping.Load() {
				break out
			}
			func() {
				s.workers.Add(1)
				s.m.EmitGauge("mimo.scheduler.workers.active.count", int64(s.workers.Load()), nil)
				defer func() {
					s.workers.Add(-1)
					s.m.EmitGauge("mimo.scheduler.workers.active.count", int64(s.workers.Load()), nil)
				}()
				// Run each process in the background context so that completion
				// of the current loop is finished before we exit cleanly, as
				// the outer process will wait for s.workers to become 0.
				_, err := a.Process(context.Background())
				if err != nil {
					log.Error(err)
				}
			}()
		case <-stop:
			break out
		}
	}
}
