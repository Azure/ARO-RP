package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
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

	lastChangefeed atomic.Value //time.Time
	startTime      time.Time
}

func NewService(log *logrus.Entry, dialer proxy.Dialer, dbOpenShiftClusters database.OpenShiftClusters, dbMaintenanceManifests database.MaintenanceManifests, m metrics.Emitter) Runnable {
	s := &service{
		baseLog: log,
		dialer:  dialer,

		dbOpenShiftClusters: dbOpenShiftClusters,

		m:           m,
		serviceName: "actuator",
		stopping:    &atomic.Bool{},
		workers:     &atomic.Int32{},

		startTime: time.Now(),
	}
	s.cond = sync.NewCond(&s.mu)
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

	for {
		s.mu.Lock()
		for s.workers.Load() >= maxWorkers && !s.stopping.Load() {
			s.cond.Wait()
		}
		s.mu.Unlock()

		if s.stopping.Load() {
			break
		}

		ocbDidWork, err := s.try(ctx)
		if err != nil {
			s.baseLog.Error(err)
		}

		if !(ocbDidWork) {
			<-t.C
		}
	}

	if !s.env.FeatureIsSet(env.FeatureDisableReadinessDelay) {
		s.waitForWorkerCompletion()
	}
	s.baseLog.Print("exiting")
	close(done)
	return nil
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
