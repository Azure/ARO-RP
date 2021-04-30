package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

const deletionTimeSetSentinel = 123456789

type Checker struct {
	openshiftClusterDocuments []*api.OpenShiftClusterDocument
	subscriptionDocuments     []*api.SubscriptionDocument
	billingDocuments          []*api.BillingDocument
	asyncOperationDocuments   []*api.AsyncOperationDocument
	portalDocuments           []*api.PortalDocument
	gatewayDocuments          []*api.GatewayDocument
}

func NewChecker() *Checker {
	return &Checker{}
}

func (f *Checker) AddOpenShiftClusterDocuments(docs ...*api.OpenShiftClusterDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.openshiftClusterDocuments = append(f.openshiftClusterDocuments, docCopy.(*api.OpenShiftClusterDocument))
	}
}

func (f *Checker) AddSubscriptionDocuments(docs ...*api.SubscriptionDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.subscriptionDocuments = append(f.subscriptionDocuments, docCopy.(*api.SubscriptionDocument))
	}
}

func (f *Checker) AddBillingDocuments(docs ...*api.BillingDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.billingDocuments = append(f.billingDocuments, docCopy.(*api.BillingDocument))
	}
}

func (f *Checker) AddAsyncOperationDocuments(docs ...*api.AsyncOperationDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.asyncOperationDocuments = append(f.asyncOperationDocuments, docCopy.(*api.AsyncOperationDocument))
	}
}

func (f *Checker) AddPortalDocuments(docs ...*api.PortalDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.portalDocuments = append(f.portalDocuments, docCopy.(*api.PortalDocument))
	}
}

func (f *Checker) AddGatewayDocuments(docs ...*api.GatewayDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.gatewayDocuments = append(f.gatewayDocuments, docCopy.(*api.GatewayDocument))
	}
}

func (f *Checker) CheckOpenShiftClusters(openShiftClusters *cosmosdb.FakeOpenShiftClusterDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := openShiftClusters.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	if len(f.openshiftClusterDocuments) != 0 && len(all.OpenShiftClusterDocuments) == len(f.openshiftClusterDocuments) {
		diff := deep.Equal(all.OpenShiftClusterDocuments, f.openshiftClusterDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.OpenShiftClusterDocuments) != 0 || len(f.openshiftClusterDocuments) != 0 {
		errs = append(errs, fmt.Errorf("openShiftClusters length different, %d vs %d", len(all.OpenShiftClusterDocuments), len(f.openshiftClusterDocuments)))
	}

	return errs
}

func (f *Checker) CheckBilling(billing *cosmosdb.FakeBillingDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := billing.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	// If they exist, change certain values to magic ones
	for _, doc := range all.BillingDocuments {
		if doc.Billing.DeletionTime != 0 {
			doc.Billing.DeletionTime = deletionTimeSetSentinel
		}
	}
	for _, doc := range f.billingDocuments {
		if doc.Billing.DeletionTime != 0 {
			doc.Billing.DeletionTime = deletionTimeSetSentinel
		}
	}

	if len(f.billingDocuments) != 0 && len(all.BillingDocuments) == len(f.billingDocuments) {
		diff := deep.Equal(all.BillingDocuments, f.billingDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.BillingDocuments) != 0 || len(f.billingDocuments) != 0 {
		errs = append(errs, fmt.Errorf("billing length different, %d vs %d", len(all.BillingDocuments), len(f.billingDocuments)))
	}

	return errs
}

func (f *Checker) CheckSubscriptions(subscriptions *cosmosdb.FakeSubscriptionDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := subscriptions.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	if len(f.subscriptionDocuments) != 0 && len(all.SubscriptionDocuments) == len(f.subscriptionDocuments) {
		diff := deep.Equal(all.SubscriptionDocuments, f.subscriptionDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.SubscriptionDocuments) != 0 || len(f.subscriptionDocuments) != 0 {
		errs = append(errs, fmt.Errorf("subscriptions length different, %d vs %d", len(all.SubscriptionDocuments), len(f.subscriptionDocuments)))
	}

	return errs
}

func (f *Checker) CheckAsyncOperations(asyncOperations *cosmosdb.FakeAsyncOperationDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := asyncOperations.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	if len(f.asyncOperationDocuments) != 0 && len(all.AsyncOperationDocuments) == len(f.asyncOperationDocuments) {
		diff := deep.Equal(all.AsyncOperationDocuments, f.asyncOperationDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.AsyncOperationDocuments) != 0 || len(f.asyncOperationDocuments) != 0 {
		errs = append(errs, fmt.Errorf("asyncOperations length different, %d vs %d", len(all.AsyncOperationDocuments), len(f.asyncOperationDocuments)))
	}

	return errs
}

func (f *Checker) CheckPortals(portals *cosmosdb.FakePortalDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := portals.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	if len(f.portalDocuments) != 0 && len(all.PortalDocuments) == len(f.portalDocuments) {
		diff := deep.Equal(all.PortalDocuments, f.portalDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.PortalDocuments) != 0 || len(f.portalDocuments) != 0 {
		errs = append(errs, fmt.Errorf("portals length different, %d vs %d", len(all.PortalDocuments), len(f.portalDocuments)))
	}

	return errs
}

func (f *Checker) CheckGateways(gateways *cosmosdb.FakeGatewayDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := gateways.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	if len(f.gatewayDocuments) != 0 && len(all.GatewayDocuments) == len(f.gatewayDocuments) {
		diff := deep.Equal(all.GatewayDocuments, f.gatewayDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.GatewayDocuments) != 0 || len(f.gatewayDocuments) != 0 {
		errs = append(errs, fmt.Errorf("gateways length different, %d vs %d", len(all.GatewayDocuments), len(f.gatewayDocuments)))
	}

	return errs
}
