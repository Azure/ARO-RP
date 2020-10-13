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
}

func NewChecker() *Checker {
	return &Checker{}
}

func (f *Checker) AddOpenShiftClusterDocuments(docs []*api.OpenShiftClusterDocument) {
	f.openshiftClusterDocuments = append(f.openshiftClusterDocuments, docs...)
}

func (f *Checker) AddOpenShiftClusterDocument(doc *api.OpenShiftClusterDocument) {
	f.openshiftClusterDocuments = append(f.openshiftClusterDocuments, doc)
}

func (f *Checker) AddSubscriptionDocuments(docs []*api.SubscriptionDocument) {
	f.subscriptionDocuments = append(f.subscriptionDocuments, docs...)
}

func (f *Checker) AddSubscriptionDocument(doc *api.SubscriptionDocument) {
	f.subscriptionDocuments = append(f.subscriptionDocuments, doc)
}

func (f *Checker) AddBillingDocuments(docs []*api.BillingDocument) {
	f.billingDocuments = append(f.billingDocuments, docs...)
}

func (f *Checker) AddBillingDocument(doc *api.BillingDocument) {
	f.billingDocuments = append(f.billingDocuments, doc)
}

func (f *Checker) AddAsyncOperationDocuments(docs []*api.AsyncOperationDocument) {
	f.asyncOperationDocuments = append(f.asyncOperationDocuments, docs...)
}

func (f *Checker) CheckOpenShiftClusters(openShiftClusters *cosmosdb.FakeOpenShiftClusterDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := openShiftClusters.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	if len(f.openshiftClusterDocuments) != 0 && len(all.OpenShiftClusterDocuments) == len(f.openshiftClusterDocuments) {
		diff := deep.Equal(all.OpenShiftClusterDocuments, f.openshiftClusterDocuments)
		if diff != nil {
			for _, i := range diff {
				errs = append(errs, errors.New(i))
			}
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
		if diff != nil {
			for _, i := range diff {
				errs = append(errs, errors.New(i))
			}
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
		if diff != nil {
			for _, i := range diff {
				errs = append(errs, errors.New(i))
			}
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
		if diff != nil {
			for _, i := range diff {
				errs = append(errs, errors.New(i))
			}
		}
	} else if len(all.AsyncOperationDocuments) != 0 || len(f.asyncOperationDocuments) != 0 {
		errs = append(errs, fmt.Errorf("asyncOperations length different, %d vs %d", len(all.AsyncOperationDocuments), len(f.asyncOperationDocuments)))
	}

	return errs
}
