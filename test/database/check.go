package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
)

const DELETION_TIME_SET = 123456789

type Checker struct {
	openshiftClusterDocuments []*api.OpenShiftClusterDocument
	subscriptionDocuments     []*api.SubscriptionDocument
	billingDocuments          []*api.BillingDocument
	asyncOperationDocuments   []*api.AsyncOperationDocument

	clients *FakeClients
}

func NewChecker(clients *FakeClients) *Checker {
	return &Checker{clients: clients}
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

func (f *Checker) Check() (errs []error) {
	for _, err := range f.CheckAsyncOperations() {
		errs = append(errs, err)
	}
	for _, err := range f.CheckOpenShiftCluster() {
		errs = append(errs, err)
	}
	for _, err := range f.CheckBilling() {
		errs = append(errs, err)
	}
	return errs
}

func (f *Checker) CheckAsyncOperations() []error {
	var errs []error
	ctx := context.Background()

	allAsyncDocs, err := f.clients.AsyncOperations.ListAll(ctx, nil)
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

func (f *Checker) CheckOpenShiftCluster() []error {
	var errs []error
	ctx := context.Background()

	// OpenShiftCluster
	allOpenShiftDocs, err := f.clients.OpenShiftClusters.ListAll(ctx, nil)
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
func (f *Checker) CheckBilling() []error {
	var errs []error
	ctx := context.Background()

	// Billing
	allBilling, err := f.clients.Billing.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	// If they exist, change certain values to magic ones
	for _, doc := range allBilling.BillingDocuments {
		if doc.Billing.DeletionTime != 0 {
			doc.Billing.DeletionTime = DELETION_TIME_SET
		}
	}

	if len(f.billingDocuments) != 0 && len(allBilling.BillingDocuments) == len(f.billingDocuments) {
		diff := deep.Equal(allBilling.BillingDocuments, f.billingDocuments)
		if diff != nil {
			for _, i := range diff {
				errs = append(errs, errors.New(i))
			}
		}
	} else if len(allBilling.BillingDocuments) != 0 || len(f.billingDocuments) != 0 {
		errs = append(errs, fmt.Errorf("billing length different, %d vs %d", len(allBilling.BillingDocuments), len(f.billingDocuments)))
	}

	return errs
}
