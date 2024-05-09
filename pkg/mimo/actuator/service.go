package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/buckets"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

const maxWorkers = 5

type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{}) error
}

type service struct {
	dialer  proxy.Dialer
	baseLog *logrus.Entry
	env     env.Interface

	dbOpenShiftClusters    database.OpenShiftClusters
	dbMaintenanceManifests database.MaintenanceManifests
	serviceName            string

	m        metrics.Emitter
	mu       sync.RWMutex
	cond     *sync.Cond
	stopping *atomic.Bool
	workers  *atomic.Int32

	b buckets.BucketWorker

	lastChangefeed atomic.Value //time.Time
	startTime      time.Time
}

func NewService(log *logrus.Entry, dialer proxy.Dialer, dbOpenShiftClusters database.OpenShiftClusters, dbMaintenanceManifests database.MaintenanceManifests, m metrics.Emitter) Runnable {
	s := &service{
		baseLog: log,
		dialer:  dialer,

		dbOpenShiftClusters:    dbOpenShiftClusters,
		dbMaintenanceManifests: dbMaintenanceManifests,

		m:           m,
		serviceName: "actuator",
		stopping:    &atomic.Bool{},
		workers:     &atomic.Int32{},

		startTime: time.Now(),
	}

	s.b = buckets.NewBucketWorker(log, s.worker, &s.mu)

	return s
}

func (s *service) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) error {
	defer recover.Panic(s.baseLog)

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
	go heartbeat.EmitHeartbeat(s.baseLog, s.m, s.serviceName+".heartbeat", nil, s.checkReady)

	lastGotDocs := make(map[string]*api.OpenShiftClusterDocument)

	for {
		if s.stopping.Load() {
			break
		}

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

func (s *service) poll(ctx context.Context, oldDocs map[string]*api.OpenShiftClusterDocument) (map[string]*api.OpenShiftClusterDocument, error) {
	i, err := s.dbOpenShiftClusters.GetAllUUIDs(ctx, "")
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

	return docMap, nil
}

func (s *service) waitForWorkerCompletion() {
	s.mu.Lock()
	for s.workers.Load() > 0 {
		s.cond.Wait()
	}
	s.mu.Unlock()
}

// checkReady checks the ready status of the monitor to make it consistent
// across the /healthz/ready endpoint and emitted metrics.   We wait for 2
// minutes before indicating health.  This ensures that there will be a gap in
// our health metric if we crash or restart.
func (s *service) checkReady() bool {
	lastChangefeedTime, ok := s.lastChangefeed.Load().(time.Time)
	if !ok {
		return false
	}
	return (time.Since(lastChangefeedTime) < time.Minute) && // did we process the change feed recently?
		(time.Since(s.startTime) > 2*time.Minute) // are we running for at least 2 minutes?
}

func (s *service) worker(stop <-chan struct{}, delay time.Duration, id string) {
	defer recover.Panic(s.baseLog)

	time.Sleep(delay)

	log := utillog.EnrichWithResourceID(s.baseLog, id)

	a, err := NewActuator(context.Background(), s.env, log, id, s.dbOpenShiftClusters, s.dbMaintenanceManifests)
	if err != nil {
		log.Error(err)
		return
	}

	t := time.NewTicker(time.Minute)
	defer t.Stop()

out:
	for {
		func() {
			s.workers.Add(1)
			s.m.EmitGauge("mimo.actuator.workers.active.count", int64(s.workers.Load()), nil)

			defer func() {
				s.workers.Add(-1)
				s.m.EmitGauge("mimo.actuator.workers.active.count", int64(s.workers.Load()), nil)
				s.cond.Signal()
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
