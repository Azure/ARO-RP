package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
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

	m              metrics.Emitter
	metricPrefix   string
	tokenRefreshed bool
}

func NewRefresher(log *logrus.Entry, env env.Core, authorizer autorest.Authorizer, insecureSkipVerify bool, dbc cosmosdb.DatabaseClient, m metrics.Emitter, url string) Refresher {
	component := strings.ToLower(env.Component())

	return &refresher{
		log: log,
		c:   NewClient(env, authorizer, insecureSkipVerify, url),

		dbc:        dbc,
		permission: component,

		m:            m,
		metricPrefix: component,
	}
}

func (r *refresher) checkRefreshAndReset() bool {
	if r.tokenRefreshed {
		r.tokenRefreshed = false
		return true
	}
	return false
}

func (r *refresher) Run(ctx context.Context) error {
	defer utilrecover.Panic(r.log)

	go heartbeat.EmitHeartbeat(r.log, r.m, r.metricPrefix+".dbtokenrefresh", nil, r.checkRefreshAndReset)

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		err := r.runOnce(ctx)
		if err != nil {
			r.log.Error(err)
		} else {
			r.lastRefresh.Store(time.Now())
			r.tokenRefreshed = true
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
