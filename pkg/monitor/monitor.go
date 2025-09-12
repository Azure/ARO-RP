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
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
)

type monitorDBs interface {
	database.DatabaseGroupWithMonitors
	database.DatabaseGroupWithOpenShiftClusters
	database.DatabaseGroupWithSubscriptions
}

type monitor struct {
	baseLog *logrus.Entry
	dialer  proxy.Dialer

	dbGroup monitorDBs

	m        metrics.Emitter
	clusterm metrics.Emitter
	mu       sync.RWMutex
	docs     map[string]*cacheDoc
	subs     map[string]*subscriptionInfo
	env      env.Interface

	isMaster    bool
	bucketCount int
	buckets     map[int]struct{}

	lastBucketlist atomic.Value //time.Time
	lastChangefeed atomic.Value //time.Time
	startTime      time.Time

	hiveClusterManagers map[int]hive.ClusterManager
}

// subscriptionInfo stores TenantID for a given subscription. We don't store the
// state as we filter out unwanted states in the changefeed.
type subscriptionInfo struct {
	TenantID string
}

type Runnable interface {
	Run(context.Context) error
}

func NewMonitor(log *logrus.Entry, dialer proxy.Dialer, dbGroup monitorDBs, m, clusterm metrics.Emitter, e env.Interface) Runnable {
	return &monitor{
		baseLog: log,
		dialer:  dialer,

		dbGroup: dbGroup,

		m:        m,
		clusterm: clusterm,
		docs:     map[string]*cacheDoc{},
		subs:     map[string]*subscriptionInfo{},
		env:      e,

		bucketCount: bucket.Buckets,
		buckets:     map[int]struct{}{},

		startTime: time.Now(),

		hiveClusterManagers: map[int]hive.ClusterManager{},
	}
}

func (mon *monitor) Run(ctx context.Context) error {
	dbMonitors, err := mon.dbGroup.Monitors()
	if err != nil {
		return err
	}

	// Load the Hive ClusterManager if configured -- NewFromEnvClusterManager
	// returns nil and no error if Hive is disabled
	cl, err := hive.NewFromEnvCLusterManager(ctx, mon.baseLog, mon.env)
	if err != nil {
		mon.baseLog.Error("failed to create Hive ClusterManager: %w", err)
		return err
	}
	if cl != nil {
		// We only have one shard
		mon.hiveClusterManagers[1] = cl
	}

	_, err = dbMonitors.Create(ctx, &api.MonitorDocument{
		ID: "master",
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) {
		return err
	}

	// fill the cache from the database change feed
	go mon.changefeed(ctx, mon.baseLog.WithField("component", "changefeed"), nil)
	go mon.changefeedMetrics(nil)

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	go heartbeat.EmitHeartbeat(mon.baseLog, mon.m, "monitor.heartbeat", nil, mon.checkReady)

	for {
		// register ourself as a monitor
		err = dbMonitors.MonitorHeartbeat(ctx)
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
