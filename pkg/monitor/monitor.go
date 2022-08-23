package monitor

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
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
)

type monitor struct {
	baseLog *logrus.Entry
	dialer  proxy.Dialer

	dbMonitors          database.Monitors
	dbOpenShiftClusters database.OpenShiftClusters
	dbSubscriptions     database.Subscriptions

	m        metrics.Emitter
	clusterm metrics.Emitter
	mu       sync.RWMutex
	docs     map[string]*cacheDoc
	subs     map[string]*api.SubscriptionDocument

	isMaster    bool
	bucketCount int
	buckets     map[int]struct{}

	lastBucketlist atomic.Value //time.Time
	lastChangefeed atomic.Value //time.Time
	startTime      time.Time

	liveConfig liveconfig.Manager
}

type Runnable interface {
	Run(context.Context) error
}

func NewMonitor(log *logrus.Entry, dialer proxy.Dialer, dbMonitors database.Monitors, dbOpenShiftClusters database.OpenShiftClusters, dbSubscriptions database.Subscriptions, m, clusterm metrics.Emitter, liveConfig liveconfig.Manager) Runnable {
	return &monitor{
		baseLog: log,
		dialer:  dialer,

		dbMonitors:          dbMonitors,
		dbOpenShiftClusters: dbOpenShiftClusters,
		dbSubscriptions:     dbSubscriptions,

		m:        m,
		clusterm: clusterm,
		docs:     map[string]*cacheDoc{},
		subs:     map[string]*api.SubscriptionDocument{},

		bucketCount: bucket.Buckets,
		buckets:     map[int]struct{}{},

		startTime: time.Now(),

		liveConfig: liveConfig,
	}
}

func (mon *monitor) Run(ctx context.Context) error {
	_, err := mon.dbMonitors.Create(ctx, &api.MonitorDocument{
		ID: "master",
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) {
		return err
	}

	// fill the cache from the database change feed
	go mon.changefeed(ctx, mon.baseLog.WithField("component", "changefeed"), nil)

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	go heartbeat.EmitHeartbeat(mon.baseLog, mon.m, "monitor.heartbeat", nil, mon.checkReady)

	for {
		// register ourself as a monitor
		err = mon.dbMonitors.MonitorHeartbeat(ctx)
		if err != nil {
			mon.baseLog.Error(err)
		}

		// try to become master and share buckets across registered monitors
		err = mon.master(ctx)
		if err != nil {
			mon.baseLog.Error(err)
		}

		// read our bucket allocation from the master
		err = mon.listBuckets(ctx)
		if err != nil {
			mon.baseLog.Error(err)
		} else {
			mon.lastBucketlist.Store(time.Now())
		}

		<-t.C
	}
}

// checkReady checks the ready status of the monitor to make it consistent
// across the /healthz/ready endpoint and emitted metrics.   We wait for 2
// minutes before indicating health.  This ensures that there will be a gap in
// our health metric if we crash or restart.
func (mon *monitor) checkReady() bool {
	lastBucketTime, ok := mon.lastBucketlist.Load().(time.Time)
	if !ok {
		return false
	}
	lastChangefeedTime, ok := mon.lastChangefeed.Load().(time.Time)
	if !ok {
		return false
	}
	return (time.Since(lastBucketTime) < time.Minute) && // did we list buckets successfully recently?
		(time.Since(lastChangefeedTime) < time.Minute) && // did we process the change feed recently?
		(time.Since(mon.startTime) > 2*time.Minute) // are we running for at least 2 minutes?
}
