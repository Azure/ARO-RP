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
	utilrecover "github.com/Azure/ARO-RP/pkg/util/recover"
)

type Refresher interface {
	Run(context.Context) error
	Ready() bool
}

type refresher struct {
	log *logrus.Entry
	c   Client

	dbc        cosmosdb.DatabaseClient
	permission string

	lastRefresh atomic.Value //time.Time
}

func NewRefresher(log *logrus.Entry, env env.Core, authorizer autorest.Authorizer, insecureSkipVerify bool, dbc cosmosdb.DatabaseClient, permission string) (Refresher, error) {
	c, err := NewClient(env, authorizer, insecureSkipVerify)
	if err != nil {
		return nil, err
	}

	return &refresher{
		log: log,
		c:   c,

		dbc:        dbc,
		permission: permission,
	}, nil
}

func (r *refresher) Run(ctx context.Context) error {
	defer utilrecover.Panic(r.log)

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

func (r *refresher) Ready() bool {
	lastRefresh, _ := r.lastRefresh.Load().(time.Time)
	return time.Since(lastRefresh) < time.Hour
}
