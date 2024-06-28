package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	maxWorkers      = 100
	maxDequeueCount = 5
)

type backend struct {
	baseLog *logrus.Entry
	env     env.Interface

	dbAsyncOperations                  database.AsyncOperations
	dbBilling                          database.Billing
	dbGateway                          database.Gateway
	dbOpenShiftClusters                database.OpenShiftClusters
	dbSubscriptions                    database.Subscriptions
	dbOpenShiftVersions                database.OpenShiftVersions
	dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets

	aead    encryption.AEAD
	m       metrics.Emitter
	billing billing.Manager

	mu       sync.Mutex
	cond     *sync.Cond
	workers  int32
	stopping atomic.Value

	ocb *openShiftClusterBackend
	sb  *subscriptionBackend
}

// Runnable represents a runnable object
type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{})
}

// NewBackend returns a new runnable backend
func NewBackend(ctx context.Context, log *logrus.Entry, env env.Interface, dbAsyncOperations database.AsyncOperations, dbBilling database.Billing, dbGateway database.Gateway, dbOpenShiftClusters database.OpenShiftClusters, dbSubscriptions database.Subscriptions, dbOpenShiftVersions database.OpenShiftVersions, dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets, aead encryption.AEAD, m metrics.Emitter) (Runnable, error) {
	b, err := newBackend(ctx, log, env, dbAsyncOperations, dbBilling, dbGateway, dbOpenShiftClusters, dbSubscriptions, dbOpenShiftVersions, dbPlatformWorkloadIdentityRoleSets, aead, m)
	if err != nil {
		return nil, err
	}

	b.ocb = newOpenShiftClusterBackend(b)
	b.sb = newSubscriptionBackend(b)
	return b, nil
}

func newBackend(ctx context.Context, log *logrus.Entry, env env.Interface, dbAsyncOperations database.AsyncOperations, dbBilling database.Billing, dbGateway database.Gateway, dbOpenShiftClusters database.OpenShiftClusters, dbSubscriptions database.Subscriptions, dbOpenShiftVersions database.OpenShiftVersions, dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets, aead encryption.AEAD, m metrics.Emitter) (*backend, error) {
	billing, err := billing.NewManager(env, dbBilling, dbSubscriptions, log)
	if err != nil {
		return nil, err
	}

	b := &backend{
		baseLog: log,
		env:     env,

		dbAsyncOperations:                  dbAsyncOperations,
		dbBilling:                          dbBilling,
		dbGateway:                          dbGateway,
		dbOpenShiftClusters:                dbOpenShiftClusters,
		dbSubscriptions:                    dbSubscriptions,
		dbOpenShiftVersions:                dbOpenShiftVersions,
		dbPlatformWorkloadIdentityRoleSets: dbPlatformWorkloadIdentityRoleSets,

		billing: billing,
		aead:    aead,
		m:       m,
	}
	b.cond = sync.NewCond(&b.mu)
	b.stopping.Store(false)
	return b, nil
}

func (b *backend) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) {
	defer recover.Panic(b.baseLog)

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	if stop != nil {
		go func() {
			defer recover.Panic(b.baseLog)

			<-stop
			b.baseLog.Print("stopping")
			b.stopping.Store(true)
			b.cond.Signal()
		}()
	}

	for {
		b.mu.Lock()
		for atomic.LoadInt32(&b.workers) >= maxWorkers && !b.stopping.Load().(bool) {
			b.cond.Wait()
		}
		b.mu.Unlock()

		if b.stopping.Load().(bool) {
			break
		}

		ocbDidWork, err := b.ocb.try(ctx)
		if err != nil {
			b.baseLog.Error(err)
		}

		sbDidWork, err := b.sb.try(ctx)
		if err != nil {
			b.baseLog.Error(err)
		}

		if !(ocbDidWork || sbDidWork) {
			<-t.C
		}
	}

	if !b.env.FeatureIsSet(env.FeatureDisableReadinessDelay) {
		b.waitForWorkerCompletion()
	}
	b.baseLog.Print("exiting")
	close(done)
}

func (b *backend) waitForWorkerCompletion() {
	b.mu.Lock()
	for atomic.LoadInt32(&b.workers) > 0 {
		b.cond.Wait()
	}
	b.mu.Unlock()
}
