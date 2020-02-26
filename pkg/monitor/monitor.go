package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
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
)

type monitor struct {
	baseLog  *logrus.Entry
	env      env.Interface
	db       *database.Database
	m        metrics.Interface
	clusterm metrics.Interface
	mu       sync.Mutex
	docs     sync.Map

	isMaster    bool
	bucketCount int
	buckets     map[int]struct{}

	ch chan string
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

		bucketCount: bucket.Buckets,
		buckets:     map[int]struct{}{},

		ch: make(chan string),
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

	// schedule work across the workers
	go mon.schedule(ctx, mon.baseLog.WithField("component", "schedule"), nil)

	// populate the workers
	for i := 0; i < 100; i++ {
		go mon.worker(ctx, mon.baseLog.WithField("component", fmt.Sprintf("worker-%d", i)))
	}

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

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
