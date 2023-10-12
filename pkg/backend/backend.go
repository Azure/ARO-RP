package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/service"
)

const (
	maxWorkers      = 100
	maxDequeueCount = 5
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

	aead    encryption.AEAD
	m       metrics.Emitter
	billing billing.Manager

	ocb *openShiftClusterBackend
	sb  *subscriptionBackend
}

// NewBackend returns a new runnable backend
func NewBackend(ctx context.Context, log *logrus.Entry, env env.Interface, dbAsyncOperations database.AsyncOperations, dbBilling database.Billing, dbGateway database.Gateway, dbOpenShiftClusters database.OpenShiftClusters, dbSubscriptions database.Subscriptions, dbOpenShiftVersions database.OpenShiftVersions, aead encryption.AEAD, m metrics.Emitter) (service.Runnable, service.Runnable, error) {
	b, err := newBackend(ctx, log, env, dbAsyncOperations, dbBilling, dbGateway, dbOpenShiftClusters, dbSubscriptions, dbOpenShiftVersions, aead, m)
	if err != nil {
		return nil, nil, err
	}
	b.ocb = newOpenShiftClusterBackend(b)
	b.ocb.q = service.NewWorkerQueue(ctx, log, env, b.ocb.try)
	b.sb = newSubscriptionBackend(b)
	b.sb.q = service.NewWorkerQueue(ctx, log, env, b.sb.try)
	return b.ocb.q, b.sb.q, nil
}

func newBackend(ctx context.Context, log *logrus.Entry, env env.Interface, dbAsyncOperations database.AsyncOperations, dbBilling database.Billing, dbGateway database.Gateway, dbOpenShiftClusters database.OpenShiftClusters, dbSubscriptions database.Subscriptions, dbOpenShiftVersions database.OpenShiftVersions, aead encryption.AEAD, m metrics.Emitter) (*backend, error) {
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

		billing: billing,
		aead:    aead,
		m:       m,
	}

	return b, nil
}
