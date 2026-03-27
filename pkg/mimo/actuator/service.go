package actuator

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
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/buckets"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{}) error
}

type service struct {
	dialer  proxy.Dialer
	baseLog *logrus.Entry
	env     env.Interface

	dbGroup actuatorDBs

	m              metrics.Emitter
	mu             sync.RWMutex
	cond           *sync.Cond
	stopping       *atomic.Bool
	workers        *atomic.Int32
	workerRoutines sync.WaitGroup

	b           buckets.BucketWorker[*api.OpenShiftClusterDocument]
	bucketCount int

	changefeedBatchSize            int
	changefeedInterval             time.Duration
	changefeedReadinessInterval    time.Duration
	taskPollTime                   time.Duration
	bucketRefreshInterval          time.Duration
	bucketRefreshReadinessInterval time.Duration

	lastChangefeed   atomic.Value // time.Time
	lastBucketUpdate atomic.Value // time.Time
	startTime        time.Time

	now         func() time.Time
	workerDelay func() time.Duration
	readyDelay  time.Duration

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask

	serveHealthz bool
}

var _ Runnable = (*service)(nil)

type actuatorDBs interface {
	database.DatabaseGroupWithOpenShiftClusters
	database.DatabaseGroupWithMaintenanceManifests
	database.DatabaseGroupWithPoolWorkers
}

func NewService(env env.Interface, log *logrus.Entry, dialer proxy.Dialer, dbg actuatorDBs, m metrics.Emitter) *service {
	s := &service{
		env:     env,
		baseLog: log,
		dialer:  dialer,

		dbGroup: dbg,

		m:           m,
		stopping:    &atomic.Bool{},
		workers:     &atomic.Int32{},
		bucketCount: bucket.Buckets,

		startTime:   time.Now(),
		workerDelay: func() time.Duration { return time.Duration(rand.Intn(60)) * time.Second },
		now:         time.Now,

		// For the OpenShiftClusterDocument polling we temporarily use a query
		// which retrieves ID and bucket rather than polling an incremental
		// feed. This means that we don't need to worry about a large batch size
		// being a huge amount of mostly-unneeded JSON (since you can't filter a
		// changefeed) or needing to align with the deletion timer like the
		// Monitor.
		changefeedBatchSize:         200,
		changefeedInterval:          10 * time.Minute,
		changefeedReadinessInterval: 12 * time.Minute,

		// The polling time for MaintenanceManifests is kept lower because we
		// prioritise responsiveness
		taskPollTime: 90 * time.Second,

		// Bucket timing is set lower to prioritise responsiveness to VM changes
		bucketRefreshInterval:          10 * time.Second,
		bucketRefreshReadinessInterval: 60 * time.Second,

		readyDelay:   time.Minute * 2,
		serveHealthz: true,
	}

	s.cond = sync.NewCond(&s.mu)
	s.b = buckets.NewBucketWorker[*api.OpenShiftClusterDocument](log, s.spawnWorker, &s.mu)
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

	t := time.NewTicker(s.changefeedInterval)
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

	// Start the bucket worker update loop which will coordinate buckets between
	// the MIMO instances
	go buckets.StartBucketWorkerLoop(
		ctx, s.baseLog, api.PoolWorkerTypeMIMOActuator,
		s.bucketCount, s.bucketRefreshInterval, dbPoolWorkers, func(i []int) {
			s.b.SetBuckets(i)
			s.lastBucketUpdate.Store(s.now())
		}, stop,
	)

	lastGotDocs := make(map[string]*api.OpenShiftClusterDocument)
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
func (s *service) poll(ctx context.Context, oldDocs map[string]*api.OpenShiftClusterDocument) (map[string]*api.OpenShiftClusterDocument, error) {
	dbOpenShiftClusters, err := s.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, err
	}

	// Fetch all of the cluster UUIDs
	i, err := dbOpenShiftClusters.GetAllResourceIDs(ctx, "")
	if err != nil {
		return nil, err
	}

	docs := make([]*api.OpenShiftClusterDocument, 0)

	for {
		d, err := i.Next(ctx, s.changefeedBatchSize)
		if err != nil {
			return nil, err
		}
		if d == nil {
			break
		}

		docs = append(docs, d.OpenShiftClusterDocuments...)
	}

	s.baseLog.Debugf("fetched %d clusters from CosmosDB", len(docs))

	docMap := make(map[string]*api.OpenShiftClusterDocument)
	for _, d := range docs {
		docMap[strings.ToLower(d.Key)] = d
	}

	// Acquire lock for when we're mutating the changefeed cache
	s.mu.Lock()
	defer s.mu.Unlock()

	// remove docs that don't exist in the new set (removed clusters)
	for oldCluster := range oldDocs {
		_, ok := docMap[strings.ToLower(oldCluster)]
		if !ok {
			s.b.DeleteDoc(oldDocs[oldCluster])
			s.baseLog.Debugf("removed %s from buckets", oldCluster)
		}
	}

	s.baseLog.Debugf("updating %d clusters", len(docMap))

	for _, cluster := range docMap {
		s.b.UpsertDoc(cluster)
	}

	// Store when we last fetched the clusters
	s.lastChangefeed.Store(s.now())

	// Emit a metric containing the size of our cache
	s.m.EmitGauge("changefeed.caches.size", int64(s.b.Size()), map[string]string{
		"name": "OpenShiftClusterDocument",
	})

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
	lastBucketUpdate, ok := s.lastBucketUpdate.Load().(time.Time)
	if !ok {
		return false
	}
	lastChangefeedTime, ok := s.lastChangefeed.Load().(time.Time)
	if !ok {
		return false
	}

	return (time.Since(lastBucketUpdate) < s.bucketRefreshReadinessInterval) && // did we list buckets successfully recently?
		(time.Since(lastChangefeedTime) < s.changefeedReadinessInterval) && // did we update our list of clusters recently?
		(time.Since(s.startTime) > s.readyDelay) // are we running for at least (the default) 2 minutes?
}

func (s *service) spawnWorker(stop <-chan struct{}, id string) {
	s.workerRoutines.Add(1)
	go s.worker(stop, id)
}

func (s *service) worker(stop <-chan struct{}, id string) {
	defer recover.Panic(s.baseLog)
	defer s.workerRoutines.Done()

	delay := s.workerDelay()
	log := utillog.EnrichWithResourceID(s.baseLog, id)
	log.Debugf("starting worker for %s in %s...", id, delay.String())

	// Wait for a randomised delay before starting
	time.Sleep(delay)

	dbOpenShiftClusters, err := s.dbGroup.OpenShiftClusters()
	if err != nil {
		log.Error(err)
		return
	}

	dbMaintenanceManifests, err := s.dbGroup.MaintenanceManifests()
	if err != nil {
		log.Error(err)
		return
	}

	a, err := NewActuator(context.Background(), s.env, log, id, dbOpenShiftClusters, dbMaintenanceManifests, s.now)
	if err != nil {
		log.Error(err)
		return
	}

	// load in the tasks for the Actuator from the controller
	a.AddMaintenanceTasks(s.tasks)

	t := time.NewTicker(s.taskPollTime)
	defer func() {
		log.Debugf("stopping worker for %s...", id)
		t.Stop()
	}()

out:
	for !s.stopping.Load() {
		func() {
			s.workers.Add(1)
			s.m.EmitGauge("mimo.actuator.workers.active.count", int64(s.workers.Load()), nil)

			defer func() {
				s.workers.Add(-1)
				s.m.EmitGauge("mimo.actuator.workers.active.count", int64(s.workers.Load()), nil)
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
