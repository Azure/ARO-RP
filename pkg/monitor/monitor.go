package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
)

type monitor struct {
	baseLog  *logrus.Entry
	env      env.Interface
	db       *database.Database
	m        metrics.Interface
	clusterm metrics.Interface
	mu       sync.RWMutex
	docs     map[string]*cacheDoc

	isMaster    bool
	bucketCount int
	buckets     map[int]struct{}

	startTime time.Time
}

type Runnable interface {
	Run(context.Context) error
}

func NewMonitor(log *logrus.Entry, env env.Interface, db *database.Database, m, clusterm metrics.Interface) Runnable {
	return &monitor{
		baseLog:  log,
		env:      env,
		db:       db,
		m:        m,
		clusterm: clusterm,
		docs:     map[string]*cacheDoc{},

		bucketCount: bucket.Buckets,
		buckets:     map[int]struct{}{},

		startTime: time.Now(),
	}
}

func (mon *monitor) Run(ctx context.Context) error {
	_, err := mon.db.Monitors.Create(ctx, &api.MonitorDocument{
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
		err = mon.db.Monitors.MonitorHeartbeat(ctx)
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
		}

		<-t.C
	}
}

// checkReady checks the ready status of the frontend to make it consistent
// across the /healthz/ready endpoint and emitted metrics.   We wait for 2
// minutes before indicating health.  This ensures that there will be a gap in
// our health metric if we crash or restart.
func (mon *monitor) checkReady() bool {
	return time.Now().Sub(mon.startTime) > 2*time.Minute
}
