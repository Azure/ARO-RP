package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/buckets"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{}) error
}

type service struct {
	dialer  proxy.Dialer
	baseLog *logrus.Entry
	env     env.Interface

	dbBucketServices       database.BucketServices
	dbOpenShiftClusters    database.OpenShiftClusters
	dbMaintenanceManifests database.MaintenanceManifests
	serviceName            string

	m            metrics.Emitter
	mu           sync.RWMutex
	b            buckets.BucketWorker
	isController bool
	stopping     *atomic.Bool

	lastChangefeed atomic.Value //time.Time
	lastBucketlist atomic.Value //time.Time
	startTime      time.Time
}

func NewService(log *logrus.Entry, dialer proxy.Dialer, dbBucketServices database.BucketServices, dbOpenShiftClusters database.OpenShiftClusters, dbMaintenanceManifests database.MaintenanceManifests, m metrics.Emitter) Runnable {
	s := &service{
		baseLog: log,
		dialer:  dialer,

		dbBucketServices:    dbBucketServices,
		dbOpenShiftClusters: dbOpenShiftClusters,

		m:           m,
		serviceName: "actuator",
		stopping:    &atomic.Bool{},

		startTime: time.Now(),
	}

	s.b = buckets.NewBucketWorker(log, s.worker, &s.mu)
	return s
}

func (s *service) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) error {
	defer recover.Panic(s.baseLog)

	_, err := s.dbBucketServices.Create(ctx, &api.BucketServiceDocument{
		ServiceName: s.serviceName,
		ServiceRole: "controller",
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) {
		return err
	}

	// fill the cache from the database change feed
	go s.changefeed(ctx, s.baseLog.WithField("component", "changefeed"), nil)

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	if stop != nil {
		go func() {
			defer recover.Panic(s.baseLog)

			<-stop
			s.baseLog.Print("stopping")
			s.stopping.Store(true)
		}()
	}

	go heartbeat.EmitHeartbeat(s.baseLog, s.m, s.serviceName+".heartbeat", nil, s.checkReady)

	for {
		if s.stopping.Load() {
			break
		}

		// register ourself as a worker
		err = s.dbBucketServices.BucketServiceHeartbeat(ctx, s.serviceName)
		if err != nil {
			s.baseLog.Error(err)
		}

		// try to become controller and share buckets across registered monitors
		err = s.controller(ctx)
		if err != nil {
			s.baseLog.Error(err)
		}

		// read our bucket allocation from the controller
		buckets, err := s.dbBucketServices.ListBuckets(ctx, s.serviceName)
		s.b.LoadBuckets(buckets)
		if err != nil {
			s.baseLog.Error(err)
		} else {
			s.lastBucketlist.Store(time.Now())
		}

		<-t.C
	}

	s.baseLog.Print("exiting")
	close(done)
	return nil
}

// checkReady checks the ready status of the monitor to make it consistent
// across the /healthz/ready endpoint and emitted metrics.   We wait for 2
// minutes before indicating health.  This ensures that there will be a gap in
// our health metric if we crash or restart.
func (s *service) checkReady() bool {
	lastBucketTime, ok := s.lastBucketlist.Load().(time.Time)
	if !ok {
		return false
	}
	lastChangefeedTime, ok := s.lastChangefeed.Load().(time.Time)
	if !ok {
		return false
	}
	return (time.Since(lastBucketTime) < time.Minute) && // did we list buckets successfully recently?
		(time.Since(lastChangefeedTime) < time.Minute) && // did we process the change feed recently?
		(time.Since(s.startTime) > 2*time.Minute) // are we running for at least 2 minutes?
}
