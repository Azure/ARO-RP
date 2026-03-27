package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/monitor/azure/nsg"
	"github.com/Azure/ARO-RP/pkg/monitor/cluster"
	hivemon "github.com/Azure/ARO-RP/pkg/monitor/hive"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/buckets"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
)

type monitorDBs interface {
	database.DatabaseGroupWithPoolWorkers
	database.DatabaseGroupWithOpenShiftClusters
	database.DatabaseGroupWithSubscriptions
}

// Defaults for the different durations. We use different values in tests to speed them up.
var (
	defaultMonitorDelay                = time.Duration(rand.Intn(60)) * time.Second
	defaultMonitorInterval             = time.Minute
	defaultMonitorReadinessDelay       = 2 * time.Minute
	defaultChangefeedInteval           = 10 * time.Second
	defaultChangefeedReadinessInterval = time.Minute
)

type monitor struct {
	baseLog *logrus.Entry
	dialer  proxy.Dialer

	dbGroup monitorDBs

	m        metrics.Emitter
	clusterm metrics.Emitter
	mu       sync.RWMutex
	clusters *clusterChangeFeedResponder
	subs     changefeed.SubscriptionsCache
	env      env.Interface

	bucketCount int

	lastBucketlist atomic.Value // time.Time
	startTime      time.Time

	hiveClusterManagers map[int]hive.ClusterManager

	clusterMonitorBuilder func(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster, env env.Interface, tenantID string, m metrics.Emitter, hourlyRun bool) (monitoring.Monitor, error)
	nsgMonitorBuilder     func(log *logrus.Entry, oc *api.OpenShiftCluster, e env.Interface, subscriptionID string, tenantID string, emitter metrics.Emitter, dims map[string]string, trigger <-chan time.Time) monitoring.Monitor
	hiveMonitorBuilder    func(log *logrus.Entry, oc *api.OpenShiftCluster, m metrics.Emitter, hourlyRun bool, hiveClusterManager hive.ClusterManager) (monitoring.Monitor, error)

	delay                   time.Duration // Time until the monitor starts running
	interval                time.Duration // Interval between monitor runs
	changefeedInterval      time.Duration // Interval between changefeed runs (updates to cluster docs)
	readyIfChangefeedWithin time.Duration // Time that the changefeed should have been changed within to be healthy
	readyDelay              time.Duration // Minimal time until the monitor will allow itself to be marked ready
}

type Runnable interface {
	Run(context.Context) error
}

func NewMonitor(log *logrus.Entry, dialer proxy.Dialer, dbGroup monitorDBs, m, clusterm metrics.Emitter, e env.Interface) Runnable {
	mon := &monitor{
		baseLog: log,
		dialer:  dialer,

		dbGroup: dbGroup,

		m:        m,
		clusterm: clusterm,
		subs:     changefeed.NewSubscriptionsChangefeedCache(m, true),
		env:      e,

		bucketCount: bucket.Buckets,

		startTime: time.Now(),

		hiveClusterManagers: map[int]hive.ClusterManager{},

		clusterMonitorBuilder: cluster.NewMonitor,
		nsgMonitorBuilder:     nsg.NewMonitor,
		hiveMonitorBuilder:    hivemon.NewHiveMonitor,

		delay:                   defaultMonitorDelay,
		interval:                defaultMonitorInterval,
		changefeedInterval:      defaultChangefeedInteval,
		readyIfChangefeedWithin: defaultChangefeedReadinessInterval,
		readyDelay:              defaultMonitorReadinessDelay,
	}

	mon.clusters = NewClusterChangefeedResponder(log, mon.worker)
	return mon
}

func (mon *monitor) Run(ctx context.Context) error {
	dbPoolWorkers, err := mon.dbGroup.PoolWorkers()
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

	err = mon.startChangefeeds(ctx, nil)
	if err != nil {
		mon.baseLog.Error(fmt.Errorf("failed to start changefeed subscriber: %w", err))
		return err
	}
	go mon.changefeedMetrics(nil)

	go heartbeat.EmitHeartbeat(mon.baseLog, mon.m, "monitor.heartbeat", nil, mon.checkReady)

	return buckets.StartBucketWorkerLoop(ctx, mon.baseLog, api.PoolWorkerTypeMonitor, mon.bucketCount, mon.changefeedInterval, dbPoolWorkers, mon.onBuckets, nil)
}

func (mon *monitor) startChangefeeds(ctx context.Context, stop <-chan struct{}) error {
	dbOpenShiftClusters, err := mon.dbGroup.OpenShiftClusters()
	if err != nil {
		return err
	}

	dbSubscriptions, err := mon.dbGroup.Subscriptions()
	if err != nil {
		return err
	}

	// fill the cache from the database change feed
	go changefeed.RunChangefeed(
		ctx, mon.baseLog.WithField("component", "changefeed"), dbOpenShiftClusters.ChangeFeed(),
		// Align this time with the deletion mechanism.
		// Go to docs/monitoring.md for the details.
		mon.changefeedInterval,
		changefeedBatchSize, mon.clusters, stop,
	)

	// fill the cache from the database change feed
	go changefeed.RunChangefeed(
		ctx, mon.baseLog.WithField("component", "changefeed"), dbSubscriptions.ChangeFeed(),
		mon.changefeedInterval,
		changefeedBatchSize, mon.subs, stop,
	)

	return nil
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
	lastClusterChangefeedTime, ok := mon.clusters.GetLastProcessed()
	if !ok {
		return false
	}
	lastSubscriptionChangefeedTime, ok := mon.subs.GetLastProcessed()
	if !ok {
		return false
	}
	return (time.Since(lastBucketTime) < mon.readyIfChangefeedWithin) && // did we list buckets successfully recently?
		(time.Since(lastClusterChangefeedTime) < mon.readyIfChangefeedWithin) && // did we process the cluster change feed recently?
		(time.Since(lastSubscriptionChangefeedTime) < mon.readyIfChangefeedWithin) && // did we process the subscription change feed recently?
		(time.Since(mon.startTime) > mon.readyDelay) // are we running for at least (the default) 2 minutes?
}
