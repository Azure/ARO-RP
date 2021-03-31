package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type Fixture struct {
	openshiftClusterDocuments []*api.OpenShiftClusterDocument
	subscriptionDocuments     []*api.SubscriptionDocument
	billingDocuments          []*api.BillingDocument
	asyncOperationDocuments   []*api.AsyncOperationDocument
	portalDocuments           []*api.PortalDocument

	openShiftClustersDatabase database.OpenShiftClusters
	billingDatabase           database.Billing
	subscriptionsDatabase     database.Subscriptions
	asyncOperationsDatabase   database.AsyncOperations
	portalDatabase            database.Portal
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

func (f *Fixture) WithPortal(db database.Portal) *Fixture {
	f.portalDatabase = db
	return f
}

func (f *Fixture) AddOpenShiftClusterDocuments(docs ...*api.OpenShiftClusterDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.openshiftClusterDocuments = append(f.openshiftClusterDocuments, docCopy.(*api.OpenShiftClusterDocument))
	}
}

func (f *Fixture) AddSubscriptionDocuments(docs ...*api.SubscriptionDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.subscriptionDocuments = append(f.subscriptionDocuments, docCopy.(*api.SubscriptionDocument))
	}
}

func (f *Fixture) AddBillingDocuments(docs ...*api.BillingDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.billingDocuments = append(f.billingDocuments, docCopy.(*api.BillingDocument))
	}
}

func (f *Fixture) AddAsyncOperationDocuments(docs ...*api.AsyncOperationDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.asyncOperationDocuments = append(f.asyncOperationDocuments, docCopy.(*api.AsyncOperationDocument))
	}
}

func (f *Fixture) AddPortalDocuments(docs ...*api.PortalDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.portalDocuments = append(f.portalDocuments, docCopy.(*api.PortalDocument))
	}
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

	for _, i := range f.portalDocuments {
		_, err := f.portalDatabase.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	return nil
}
