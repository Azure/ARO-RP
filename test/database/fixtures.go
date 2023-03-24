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
	openshiftClusterDocuments            []*api.OpenShiftClusterDocument
	subscriptionDocuments                []*api.SubscriptionDocument
	billingDocuments                     []*api.BillingDocument
	asyncOperationDocuments              []*api.AsyncOperationDocument
	portalDocuments                      []*api.PortalDocument
	gatewayDocuments                     []*api.GatewayDocument
	openShiftVersionDocuments            []*api.OpenShiftVersionDocument
	clusterManagerConfigurationDocuments []*api.ClusterManagerConfigurationDocument

	openShiftClustersDatabase            database.OpenShiftClusters
	billingDatabase                      database.Billing
	subscriptionsDatabase                database.Subscriptions
	asyncOperationsDatabase              database.AsyncOperations
	portalDatabase                       database.Portal
	gatewayDatabase                      database.Gateway
	openShiftVersionsDatabase            database.OpenShiftVersions
	clusterManagerConfigurationsDatabase database.ClusterManagerConfigurations

	openShiftVersionsUUID uuid.Generator
}

func NewFixture() *Fixture {
	return &Fixture{}
}

func (f *Fixture) WithClusterManagerConfigurations(db database.ClusterManagerConfigurations) *Fixture {
	f.clusterManagerConfigurationsDatabase = db
	return f
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

func (f *Fixture) WithGateway(db database.Gateway) *Fixture {
	f.gatewayDatabase = db
	return f
}

func (f *Fixture) WithOpenShiftVersions(db database.OpenShiftVersions, uuid uuid.Generator) *Fixture {
	f.openShiftVersionsDatabase = db
	f.openShiftVersionsUUID = uuid
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

func (f *Fixture) AddGatewayDocuments(docs ...*api.GatewayDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.gatewayDocuments = append(f.gatewayDocuments, docCopy.(*api.GatewayDocument))
	}
}

func (f *Fixture) AddOpenShiftVersionDocuments(docs ...*api.OpenShiftVersionDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.openShiftVersionDocuments = append(f.openShiftVersionDocuments, docCopy.(*api.OpenShiftVersionDocument))
	}
}

func (f *Fixture) AddClusterManagerConfigurationDocuments(docs ...*api.ClusterManagerConfigurationDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.clusterManagerConfigurationDocuments = append(f.clusterManagerConfigurationDocuments, docCopy.(*api.ClusterManagerConfigurationDocument))
	}
}

func (f *Fixture) Create() error {
	ctx := context.Background()

	for _, i := range f.openshiftClusterDocuments {
		if i.ID == "" {
			i.ID = f.openShiftClustersDatabase.NewUUID()
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

	for _, i := range f.gatewayDocuments {
		_, err := f.gatewayDatabase.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	for _, i := range f.openShiftVersionDocuments {
		if i.ID == "" {
			i.ID = f.openShiftVersionsDatabase.NewUUID()
		}
		_, err := f.openShiftVersionsDatabase.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	for _, i := range f.clusterManagerConfigurationDocuments {
		if i.ID == "" {
			i.ID = f.clusterManagerConfigurationsDatabase.NewUUID()
		}
		_, err := f.clusterManagerConfigurationsDatabase.Create(ctx, i)
		if err != nil {
			return err
		}
	}

	return nil
}
