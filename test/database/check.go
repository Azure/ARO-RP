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

const DELETION_TIME_SET = 123456789

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

func (f *Checker) AddAsyncOperationDocument(doc *api.AsyncOperationDocument) {
	f.asyncOperationDocuments = append(f.asyncOperationDocuments, doc)
}

func (f *Checker) CheckAsyncOperations(AsyncOperations *cosmosdb.FakeAsyncOperationDocumentClient) []error {
	var errs []error
	ctx := context.Background()

	allAsyncDocs, err := AsyncOperations.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	if len(f.asyncOperationDocuments) != 0 && len(allAsyncDocs.AsyncOperationDocuments) == len(f.asyncOperationDocuments) {
		diff := deep.Equal(allAsyncDocs.AsyncOperationDocuments, f.asyncOperationDocuments)
		if diff != nil {
			for _, i := range diff {
				errs = append(errs, errors.New(i))
			}
		}
	} else if len(allAsyncDocs.AsyncOperationDocuments) != 0 || len(f.asyncOperationDocuments) != 0 {
		errs = append(errs, fmt.Errorf("async docs length different, %d vs %d", len(allAsyncDocs.AsyncOperationDocuments), len(f.asyncOperationDocuments)))
	}
	return errs
}

func (f *Checker) CheckOpenShiftCluster(OpenShiftClusters *cosmosdb.FakeOpenShiftClusterDocumentClient) []error {
	var errs []error
	ctx := context.Background()

	// OpenShiftCluster
	allOpenShiftDocs, err := OpenShiftClusters.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	if len(f.openshiftClusterDocuments) != 0 && len(allOpenShiftDocs.OpenShiftClusterDocuments) == len(f.openshiftClusterDocuments) {
		diff := deep.Equal(allOpenShiftDocs.OpenShiftClusterDocuments, f.openshiftClusterDocuments)
		if diff != nil {
			for _, i := range diff {
				errs = append(errs, errors.New(i))
			}
		}
	} else if len(allOpenShiftDocs.OpenShiftClusterDocuments) != 0 || len(f.openshiftClusterDocuments) != 0 {
		errs = append(errs, fmt.Errorf("openshiftcluster length different, %d vs %d", len(allOpenShiftDocs.OpenShiftClusterDocuments), len(f.openshiftClusterDocuments)))
	}

	return errs
}

func (f *Checker) CheckBilling(Billing *cosmosdb.FakeBillingDocumentClient) []error {
	var errs []error
	ctx := context.Background()

	// Billing
	all, err := Billing.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	// If they exist, change certain values to magic ones
	for _, doc := range all.BillingDocuments {
		if doc.Billing.DeletionTime != 0 {
			doc.Billing.DeletionTime = DELETION_TIME_SET
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

func (f *Checker) CheckSubscriptions(Subscriptions *cosmosdb.FakeSubscriptionDocumentClient) []error {
	var errs []error
	ctx := context.Background()

	// Billing
	all, err := Subscriptions.ListAll(ctx, nil)
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
		errs = append(errs, fmt.Errorf("billing length different, %d vs %d", len(all.SubscriptionDocuments), len(f.subscriptionDocuments)))
	}

	return errs
}
