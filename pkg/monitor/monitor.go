package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
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
)

type monitor struct {
	baseLog *logrus.Entry
	dialer  proxy.Dialer

	dbMonitors          database.Monitors
	dbOpenShiftClusters database.OpenShiftClusters
	dbSubscriptions     database.Subscriptions

	m        metrics.Interface
	clusterm metrics.Interface
	mu       sync.RWMutex
	docs     map[string]*cacheDoc
	subs     map[string]*api.SubscriptionDocument

	isMaster    bool
	bucketCount int
	buckets     map[int]struct{}

	now              func() time.Time
	lastBucketlist   atomic.Value //time.Time
	lastChangefeed   atomic.Value //time.Time
	startTime        time.Time
	startGracePeriod time.Duration
}

type Runnable interface {
	Run(context.Context) error
}

func NewMonitor(log *logrus.Entry, dialer proxy.Dialer, dbMonitors database.Monitors, dbOpenShiftClusters database.OpenShiftClusters, dbSubscriptions database.Subscriptions, m, clusterm metrics.Interface, startGracePeriod time.Duration) Runnable {
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

		now:              time.Now,
		startTime:        time.Now(),
		startGracePeriod: startGracePeriod,
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
func (mon *monitor) checkReady() (failed bool, failing map[string]string) {
	failing = make(map[string]string)

	// did we list buckets successfully recently?
	lastBucketTime, ok := mon.lastBucketlist.Load().(time.Time)
	if !ok {
		failed = true
		failing["lastBucketTime"] = "buckets not yet read"
	} else {
		sinceLastBucketTime := mon.now().Sub(lastBucketTime)
		if sinceLastBucketTime > time.Minute {
			failed = true
			failing["lastBucketTime"] = fmt.Sprintf("running behind, %s > %s", sinceLastBucketTime, time.Minute)
		}
	}

	// did we process the change feed recently?
	lastChangefeedTime, ok := mon.lastChangefeed.Load().(time.Time)
	if !ok {
		failed = true
		failing["lastChangefeedTime"] = "changefeed not yet read"
	} else {
		sinceLastChangefeedTime := mon.now().Sub(lastChangefeedTime)
		if sinceLastChangefeedTime > time.Minute {
			failed = true
			failing["lastChangefeedTime"] = fmt.Sprintf("running behind, %s > %s", sinceLastChangefeedTime, time.Minute)
		}
	}

	// If we are in our starting grace period, return OK (but still emit
	// individual failures)
	inGracePeriod := mon.now().Sub(mon.startTime) <= mon.startGracePeriod
	if inGracePeriod {
		failed = false
	}

	return failed, failing
}
