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

	db *database.Database
}

func NewFixture(db *database.Database) *Fixture {
	return &Fixture{db: db}
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
		_, err := f.db.OpenShiftClusters.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	for _, i := range f.subscriptionDocuments {
		_, err := f.db.Subscriptions.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	for _, i := range f.billingDocuments {
		_, err := f.db.Billing.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	for _, i := range f.asyncOperationDocuments {
		_, err := f.db.AsyncOperations.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	return nil
}
