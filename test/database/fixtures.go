package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
)

type Fixture struct {
	openshiftClusterDocuments []*api.OpenShiftClusterDocument
	subscriptionDocuments     []*api.SubscriptionDocument
	billingDocuments          []*api.BillingDocument
	asyncOperationDocuments   []*api.AsyncOperationDocument

	openShiftClustersDatabase database.OpenShiftClusters
	billingDatabase           database.Billing
	subscriptionsDatabase     database.Subscriptions
	asyncOperationsDatabase   database.AsyncOperations
}

func NewFixture() *Fixture {
	return &Fixture{}
}

func (f *Fixture) WithOpenShiftClusters(db database.OpenShiftClusters) *Fixture {
	f.openShiftClustersDatabase = db
	return f
}

func (f *Fixture) WithBilling(db database.Billing) *Fixture {
	f.billingDatabase = db
	return f
}

func (f *Fixture) WithSubscriptions(db database.Subscriptions) *Fixture {
	f.subscriptionsDatabase = db
	return f
}

func (f *Fixture) WithAsyncOperations(db database.AsyncOperations) *Fixture {
	f.asyncOperationsDatabase = db
	return f
}

func (f *Fixture) AddOpenShiftClusterDocuments(docs []*api.OpenShiftClusterDocument) {
	f.openshiftClusterDocuments = append(f.openshiftClusterDocuments, docs...)
}

func (f *Fixture) AddOpenShiftClusterDocument(doc *api.OpenShiftClusterDocument) {
	f.openshiftClusterDocuments = append(f.openshiftClusterDocuments, doc)
}

func (f *Fixture) AddSubscriptionDocuments(docs []*api.SubscriptionDocument) {
	f.subscriptionDocuments = append(f.subscriptionDocuments, docs...)
}

func (f *Fixture) AddSubscriptionDocument(doc *api.SubscriptionDocument) {
	f.subscriptionDocuments = append(f.subscriptionDocuments, doc)
}

func (f *Fixture) AddBillingDocuments(docs []*api.BillingDocument) {
	f.billingDocuments = append(f.billingDocuments, docs...)
}

func (f *Fixture) AddBillingDocument(doc *api.BillingDocument) {
	f.billingDocuments = append(f.billingDocuments, doc)
}

func (f *Fixture) AddAsyncOperationDocuments(docs []*api.AsyncOperationDocument) {
	f.asyncOperationDocuments = append(f.asyncOperationDocuments, docs...)
}

func (f *Fixture) AddAsyncOperationDocument(doc *api.AsyncOperationDocument) {
	f.asyncOperationDocuments = append(f.asyncOperationDocuments, doc)
}

func (f *Fixture) Create() error {
	ctx := context.Background()

	for _, i := range f.openshiftClusterDocuments {
		if i.ID == "" {
			i.ID = uuid.NewV4().String()
		}
		_, err := f.openShiftClustersDatabase.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	for _, i := range f.subscriptionDocuments {
		_, err := f.subscriptionsDatabase.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	for _, i := range f.billingDocuments {
		_, err := f.billingDatabase.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	for _, i := range f.asyncOperationDocuments {
		_, err := f.asyncOperationsDatabase.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	return nil
}
