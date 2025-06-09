package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/proxy"
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

	m        metrics.Emitter
	mu       sync.RWMutex
	cond     *sync.Cond
	stopping *atomic.Bool
	workers  *atomic.Int32

	b buckets.BucketWorker

	lastChangefeed atomic.Value //time.Time
	startTime      time.Time

	pollTime time.Duration
	now      func() time.Time

	tasks map[string]tasks.MaintenanceTask

	serveHealthz bool
}

type actuatorDBs interface {
	database.DatabaseGroupWithOpenShiftClusters
	database.DatabaseGroupWithMaintenanceManifests
}

func NewService(env env.Interface, log *logrus.Entry, dialer proxy.Dialer, dbg actuatorDBs, m metrics.Emitter) *service {
	s := &service{
		env:     env,
		baseLog: log,
		dialer:  dialer,

		dbGroup: dbg,

		m:        m,
		stopping: &atomic.Bool{},
		workers:  &atomic.Int32{},

		startTime: time.Now(),
		now:       time.Now,
		pollTime:  time.Minute,

		serveHealthz: true,
	}

	s.b = buckets.NewBucketWorker(log, s.worker, &s.mu)

	return s
}

func (s *service) SetMaintenanceTasks(tasks map[string]tasks.MaintenanceTask) {
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
		d, err := i.Next(ctx, -1)
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

	// remove docs that don't exist in the new set (removed clusters)
	for _, oldCluster := range maps.Keys(oldDocs) {
		_, ok := docMap[strings.ToLower(oldCluster)]
		if !ok {
			s.b.DeleteDoc(oldDocs[oldCluster])
			s.baseLog.Debugf("removed %s from buckets", oldCluster)
		}
	}

	s.baseLog.Debugf("updating %d clusters", len(docMap))

	for _, cluster := range maps.Values(docMap) {
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

	if s.env.IsLocalDevelopmentMode() {
		return (time.Since(lastChangefeedTime) < time.Minute) // did we update our list of clusters recently?
	} else {
		return (time.Since(lastChangefeedTime) < time.Minute) && // did we update our list of clusters recently?
			(time.Since(s.startTime) > 2*time.Minute) // are we running for at least 2 minutes?
	}
}

func (s *service) worker(stop <-chan struct{}, delay time.Duration, id string) {
	defer recover.Panic(s.baseLog)

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

	t := time.NewTicker(s.pollTime)
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
