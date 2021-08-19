package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	utilrecover "github.com/Azure/ARO-RP/pkg/util/recover"
)

type Refresher interface {
	Run(context.Context) error
	HasSyncedOnce() bool
}

type refresher struct {
	log *logrus.Entry
	c   Client

	dbc        cosmosdb.DatabaseClient
	permission string

	lastRefresh atomic.Value //time.Time

	m            metrics.Interface
	metricPrefix string
}

func NewRefresher(log *logrus.Entry, env env.Core, authorizer autorest.Authorizer, insecureSkipVerify bool, dbc cosmosdb.DatabaseClient, permission string, m metrics.Interface, metricPrefix string) (Refresher, error) {
	c, err := NewClient(env, authorizer, insecureSkipVerify)
	if err != nil {
		return nil, err
	}

	return &refresher{
		log: log,
		c:   c,

		dbc:        dbc,
		permission: permission,

		m:            m,
		metricPrefix: metricPrefix,
	}, nil
}

func (r *refresher) Run(ctx context.Context) error {
	defer utilrecover.Panic(r.log)

	go r.metrics()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		err := r.runOnce(ctx)
		if err != nil {
			r.log.Error(err)
		} else {
			r.lastRefresh.Store(time.Now())
		}

		<-t.C
	}
}

func (r *refresher) metrics() {
	defer utilrecover.Panic(r.log)

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for {
		if lastRefresh, ok := r.lastRefresh.Load().(time.Time); ok {
			r.m.EmitGauge(r.metricPrefix+".lastrefresh", lastRefresh.Unix(), nil)
		}

		<-t.C
	}
}

func (r *refresher) runOnce(ctx context.Context) (err error) {
	// extra hardening to prevent a panic under runOnce taking out the refresher
	// goroutine
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic: %s (original err: %v)\n\n%s", e, err, string(debug.Stack()))
		}
	}()

	timeoutCtx, done := context.WithTimeout(ctx, time.Minute)
	defer done()

	token, err := r.c.Token(timeoutCtx, r.permission)
	if err != nil {
		return err
	}

	r.dbc.SetAuthorizer(cosmosdb.NewTokenAuthorizer(token))

	return nil
}

func (r *refresher) HasSyncedOnce() bool {
	_, ok := r.lastRefresh.Load().(time.Time)
	return ok
}
