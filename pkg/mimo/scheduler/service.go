package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
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
	defaultWorkerMaxStartupDelay                  = 60 * time.Second
	defaultServiceInterval                        = 15 * time.Second
	defaultReadinessDelay                         = 2 * time.Minute
	defaultSchedulePollInterval                   = 30 * time.Second
	defaultSchedulePollReadinessInterval          = 90 * time.Second
	defaultScheduleUnconditionalReconcileInterval = 60 * time.Minute
	defaultChangefeedInteval                      = 10 * time.Second
	defaultChangefeedReadinessInterval            = time.Minute
	defaultBucketRefreshInterval                  = 10 * time.Second
	defaultBucketRefreshTTL                       = 60 * time.Second
	defaultBucketRefreshReadinessInterval         = defaultBucketRefreshTTL
)

type service struct {
	baseLog     *logrus.Entry
	env         env.Interface
	randFloat64 func() float64

	dbGroup schedulerDBs

	m            metrics.Emitter
	mu           sync.RWMutex
	stopping     *atomic.Bool
	workerCount  *atomic.Int32
	newScheduler newSchedulerFunc

	buckets  atomic.Value // []int
	b        buckets.WorkerPool[*api.MaintenanceScheduleDocument]
	subs     changefeed.SubscriptionsCache
	clusters *openShiftClusterCache

	bucketCount         int
	changefeedBatchSize int

	lastScheduleUpdate atomic.Value // time.Time
	lastBucketUpdate   atomic.Value // time.Time
	startTime          time.Time

	workerMaxStartupDelay                  time.Duration // Maximum interval before a worker starts
	interval                               time.Duration // Interval between service runs
	schedulePollInterval                   time.Duration // Interval between updates to Schedules
	schedulePollReadinessInterval          time.Duration // Time that the Schedules should have been updated within to be ready
	scheduleUnconditionalReconcileInterval time.Duration // Interval between times that a Schedule is reconciled unconditionally
	changefeedInterval                     time.Duration // Interval between changefeed runs (updates to cluster docs + subscriptions)
	bucketRefreshInterval                  time.Duration
	bucketRefreshTTL                       time.Duration // TTL for worker PoolWorker documents
	bucketRefreshReadinessInterval         time.Duration
	changefeedReadinessInterval            time.Duration // Time that the changefeed should have been changed within to be healthy
	readinessDelay                         time.Duration // Minimal time until the service will allow itself to be marked ready

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask

	scheduleShouldBeReevaluated *xsync.Map[string, bool]
	scheduleLastRunTime         *xsync.Map[string, time.Time]

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
		env:         env,
		baseLog:     log,
		randFloat64: rand.Float64,

		dbGroup: dbg,

		m:           m,
		stopping:    &atomic.Bool{},
		workerCount: &atomic.Int32{},
		bucketCount: bucket.Buckets,

		startTime:             env.Now(),
		workerMaxStartupDelay: defaultWorkerMaxStartupDelay,
		newScheduler:          NewSchedulerForSchedule,

		changefeedBatchSize:                    50,
		interval:                               defaultServiceInterval,
		changefeedInterval:                     defaultChangefeedInteval,
		changefeedReadinessInterval:            defaultChangefeedReadinessInterval,
		bucketRefreshInterval:                  defaultBucketRefreshInterval,
		bucketRefreshTTL:                       defaultBucketRefreshTTL,
		bucketRefreshReadinessInterval:         defaultBucketRefreshReadinessInterval,
		readinessDelay:                         defaultReadinessDelay,
		schedulePollInterval:                   defaultSchedulePollInterval,
		schedulePollReadinessInterval:          defaultSchedulePollReadinessInterval,
		scheduleUnconditionalReconcileInterval: defaultScheduleUnconditionalReconcileInterval,

		subs: changefeed.NewSubscriptionsChangefeedCache(m, false),

		scheduleShouldBeReevaluated: xsync.NewMap[string, bool](),
		scheduleLastRunTime:         xsync.NewMap[string, time.Time](),

		serveHealthz:  true,
		emitHeartbeat: true,
	}

	s.clusters = newOpenShiftClusterCache(log, m, s.subs)
	s.b = buckets.NewWorkerPool[*api.MaintenanceScheduleDocument](log, s.worker)
	return s
}

func (s *service) SetMaintenanceTasks(tasks map[api.MIMOTaskID]tasks.MaintenanceTask) {
	s.tasks = tasks
}

func (s *service) Run(_ctx context.Context, stop <-chan struct{}, done chan<- struct{}) error {
	defer recover.Panic(s.baseLog)
	defer close(done)

	// Set up a cancel context for signalling exits (e.g. the stop channel
	// closing, bucket fetching erroring)
	ctx, cancel := context.WithCancelCause(_ctx)
	defer cancel(nil)

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
			cancel(nil)
		}()
	}

	if s.emitHeartbeat {
		go heartbeat.EmitHeartbeat(s.baseLog, s.m, "scheduler.heartbeat", stop, s.checkReady)
	}

	waitForFirstBucketUpdate := &sync.WaitGroup{}
	waitForFirstBucketUpdate.Add(1)

	// Start the bucket worker update loop which will coordinate buckets between
	// the MIMO instances
	go buckets.StartBucketRefreshLoop(
		_ctx, s.baseLog, api.PoolWorkerTypeMIMOScheduler,
		s.bucketCount, s.bucketRefreshInterval, s.bucketRefreshTTL, dbPoolWorkers, func(i []int) {
			old, ok := s.buckets.Load().([]int)
			if !ok || !slices.Equal(old, i) {
				s.buckets.Store(i)
				// If we have a bucket update, mark all schedules as needing to be
				// reevaluated by deleting the markers
				s.scheduleShouldBeReevaluated.Clear()
			}
			if len(i) > 0 {
				s.lastBucketUpdate.Store(s.env.Now())
			}
		}, stop, cancel, waitForFirstBucketUpdate,
	)

	// Wait until we have collected our buckets before starting the poll loop/changefeeds
	waitForFirstBucketUpdate.Wait()
	if ctx.Err() != nil {
		s.baseLog.Errorf("bucket worker startup failed, exiting: %s", context.Cause(ctx))
		return context.Cause(ctx)
	}

	err = s.startChangefeeds(ctx, stop)
	if err != nil {
		return err
	}

	t := time.NewTicker(s.schedulePollInterval)

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
			s.baseLog.Warnf("context closed, stopping poll loop: %s", context.Cause(ctx))
			s.stopping.Store(true)
		}
	}

	// If we're here, we're exiting
	s.baseLog.Print("exiting, waiting for all workers to finish")
	s.b.StopAndWait()
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
		ctx, s.baseLog.WithField("component", "changefeed"), s.m, "SubscriptionDocument",
		dbSubscriptions.ChangeFeed(),
		s.changefeedInterval,
		s.changefeedBatchSize, s.subs, stop,
	)

	// start cluster changefeed
	go changefeed.RunChangefeed(
		ctx, s.baseLog.WithField("component", "changefeed"), s.m, "OpenShiftClusterDocument",
		dbOpenShiftClusters.ChangeFeed(),
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

	for _, schedule := range docMap {
		oldDoc, docExists := s.b.Doc(strings.ToLower(schedule.ID))
		if !docExists {
			// If this schedule is new, reevaluate it
			s.scheduleShouldBeReevaluated.Store(strings.ToLower(schedule.ID), true)
		} else if schedule.Timestamp != oldDoc.Timestamp {
			// If the changed timestamp is different, reevaluate the schedule
			s.scheduleShouldBeReevaluated.Store(strings.ToLower(schedule.ID), true)
		}

		s.b.UpsertDoc(schedule)
	}

	// Store when we last fetched the schedules
	s.lastScheduleUpdate.Store(s.env.Now())

	// Emit a metric containing the size of our cache
	s.m.EmitGauge("changefeed.caches.size", int64(s.b.CacheSize()), map[string]string{
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

	return (now.Sub(lastScheduleUpdate) < s.schedulePollReadinessInterval && // did we update our changefeeds recently?
		now.Sub(lastClusterChangefeed) < s.changefeedReadinessInterval &&
		now.Sub(lastSubsChangefeed) < s.changefeedReadinessInterval &&
		now.Sub(lastBucketUpdate) < s.bucketRefreshReadinessInterval &&
		now.Sub(s.startTime) > s.readinessDelay) // are we running for at least (the default) 2 minutes?
}

func (s *service) worker(stop <-chan struct{}, id string) {
	defer recover.Panic(s.baseLog)

	// This determines how far offset into the startup delay as well as the
	// unconditional reconcile interval this worker will run
	delayFraction := s.randFloat64()

	delay := time.Second * time.Duration(s.workerMaxStartupDelay.Seconds()*delayFraction)
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
				bucket, ok := d.GetString(string(SelectorDataKeyBucketID))
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

out:
	for !s.stopping.Load() {
		func() {
			// Store the run time as we want to start at the same time every
			// interval, not interval+eval time
			now := s.env.Now()

			// Check if this schedule has updated. Missing the marker (e.g. when
			// buckets update) means it should be run. Replace it as false, we
			// will reset any true value on failure.
			shouldReevaluateSchedule, hasMarker := s.scheduleShouldBeReevaluated.LoadAndStore(id, false)
			// Check if it's been long enough that we should run it unconditionally anyway.
			reevaluateUnconditionally := false
			lastRunTime, hasRun := s.scheduleLastRunTime.Load(id)
			if hasRun {
				reevaluateUnconditionally = shouldReevaluateUnconditionally(
					now, lastRunTime, s.scheduleUnconditionalReconcileInterval, delayFraction)
			}

			// If we don't need to reevaluate it because the schedule/buckets
			// updated, and we aren't due to unconditionally reevaluate it, skip
			// this time.
			if (!shouldReevaluateSchedule && hasMarker) && !reevaluateUnconditionally {
				return
			}

			s.workerCount.Add(1)
			s.m.EmitGauge("mimo.scheduler.workers.active.count", int64(s.workerCount.Load()), nil)
			defer func() {
				s.workerCount.Add(-1)
				s.m.EmitGauge("mimo.scheduler.workers.active.count", int64(s.workerCount.Load()), nil)
			}()
			// Run each process in the background context so that completion
			// of the current loop is finished before we exit cleanly, as
			// the outer process will wait for s.workers to become 0.
			_, err := a.Process(context.Background())
			if err != nil {
				log.Error(err)
				// On error, reset the previous reevaluation marker if it was true
				if shouldReevaluateSchedule {
					s.scheduleShouldBeReevaluated.Store(id, shouldReevaluateSchedule)
				}
			} else {
				// Update the last scheduled run time if we succeeded, so
				// failures will retry next loop.
				s.scheduleLastRunTime.Store(id, now)
			}
		}()

		select {
		case <-t.C:
		case <-stop:
			break out
		}
	}
	log.Debugf("worker for %s finished", id)
}

func shouldReevaluateUnconditionally(now time.Time, lastRunTime time.Time, interval time.Duration, delayFraction float64) bool {
	// Add the interval (e.g. the default 60 minutes)
	shiftInterval := interval +
		// Offset this schedule delayFraction amount into the interval (e.g. so
		// a 60 minute interval and a schedule with a 0.5 delayFraction will run
		// at startup and then 90 minutes later, then every 60 minutes
		// thereafter). The microsecond is removed to make it inclusive (e.g. 00:00
		// to 59:59 for the 1hr example)
		time.Duration(float64(interval-time.Microsecond)*delayFraction) -
		// Make it now >= by removing a microsecond
		time.Microsecond
	return now.After(lastRunTime.Add(shiftInterval))
}
