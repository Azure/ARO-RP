package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
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
	tickerPeriod    = 10 // seconds

	// A backend will try to acquire the backend lease at a
	// slower period than the backend ticker by counting ticks.
	tryLeasePeriod = database.BackendsLeaseTTL / tickerPeriod
)

type backend struct {
	baseLog *logrus.Entry
	env     env.Interface

	dbAsyncOperations   database.AsyncOperations
	dbBilling           database.Billing
	dbGateway           database.Gateway
	dbOpenShiftClusters database.OpenShiftClusters
	dbSubscriptions     database.Subscriptions
	dbOpenShiftVersions database.OpenShiftVersions
	dbBackends          database.Backends

	aead    encryption.AEAD
	m       metrics.Emitter
	billing billing.Manager

	mu       sync.Mutex
	cond     *sync.Cond
	workers  int32
	stopping atomic.Value

	tryLeaseTimer int32

	ocb *openShiftClusterBackend
	sb  *subscriptionBackend
}

// Runnable represents a runnable object
type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{})
}

// NewBackend returns a new runnable backend
func NewBackend(ctx context.Context, log *logrus.Entry, env env.Interface, dbAsyncOperations database.AsyncOperations, dbBilling database.Billing, dbGateway database.Gateway, dbOpenShiftClusters database.OpenShiftClusters, dbSubscriptions database.Subscriptions, dbOpenShiftVersions database.OpenShiftVersions, dbBackends database.Backends, aead encryption.AEAD, m metrics.Emitter) (Runnable, error) {
	b, err := newBackend(ctx, log, env, dbAsyncOperations, dbBilling, dbGateway, dbOpenShiftClusters, dbSubscriptions, dbOpenShiftVersions, dbBackends, aead, m)
	if err != nil {
		return nil, err
	}

	b.ocb = newOpenShiftClusterBackend(b)
	b.sb = newSubscriptionBackend(b)
	return b, nil
}

func newBackend(ctx context.Context, log *logrus.Entry, env env.Interface, dbAsyncOperations database.AsyncOperations, dbBilling database.Billing, dbGateway database.Gateway, dbOpenShiftClusters database.OpenShiftClusters, dbSubscriptions database.Subscriptions, dbOpenShiftVersions database.OpenShiftVersions, dbBackends database.Backends, aead encryption.AEAD, m metrics.Emitter) (*backend, error) {
	billing, err := billing.NewManager(env, dbBilling, dbSubscriptions, log)
	if err != nil {
		return nil, err
	}

	b := &backend{
		baseLog: log,
		env:     env,

		dbAsyncOperations:   dbAsyncOperations,
		dbBilling:           dbBilling,
		dbGateway:           dbGateway,
		dbOpenShiftClusters: dbOpenShiftClusters,
		dbSubscriptions:     dbSubscriptions,
		dbOpenShiftVersions: dbOpenShiftVersions,
		dbBackends:          dbBackends,

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

	t := time.NewTicker(tickerPeriod * time.Second)
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

	err := b.dbBackends.Initialize(ctx)
	if err != nil {
		b.baseLog.Error(err)
	}

	var hadBackendDoc bool

	for {
		b.mu.Lock()
		for atomic.LoadInt32(&b.workers) >= maxWorkers && !b.stopping.Load().(bool) {
			b.cond.Wait()
		}
		b.mu.Unlock()

		if b.stopping.Load().(bool) {
			break
		}

		var backendDoc *api.BackendDocument
		if b.tryLeaseTimer > 0 {
			// This means we previously failed to get
			// the backend lease, so decrement a tick
			// counter to zero before trying again.
			b.tryLeaseTimer--
		} else {
			backendDoc, err = b.dbBackends.TryLease(ctx)
			if err != nil {
				b.baseLog.Error(err)
			} else if backendDoc == nil {
				// Did not get the backend lease.
				// Reset timer to try again later.
				b.tryLeaseTimer = tryLeasePeriod
			}
		}
		if backendDoc != nil && !hadBackendDoc {
			b.baseLog.Info("Got the backend lease")
		} else if backendDoc == nil && hadBackendDoc {
			b.baseLog.Info("Lost the backend lease")
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

		hadBackendDoc = (backendDoc != nil)
	}

	if !b.env.FeatureIsSet(env.FeatureDisableReadinessDelay) {
		b.waitForWorkerCompletion()
	}
	b.baseLog.Print("exiting")
	close(done)
}

func (b *backend) patchWithLease(ctx context.Context, cb func(*api.BackendDocument) error) (*api.BackendDocument, error) {
	doc, err := b.dbBackends.PatchWithLease(ctx, cb)
	if err != nil && err.Error() == "lost lease" {
		b.baseLog.Info("Lost the backend lease")
		b.tryLeaseTimer = tryLeasePeriod
		return nil, nil
	}
	return doc, err
}

func (b *backend) waitForWorkerCompletion() {
	b.mu.Lock()
	for atomic.LoadInt32(&b.workers) > 0 {
		b.cond.Wait()
	}
	b.mu.Unlock()
}
