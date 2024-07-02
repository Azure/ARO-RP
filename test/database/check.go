package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

const deletionTimeSetSentinel = 123456789

type Checker struct {
	openshiftClusterDocuments                []*api.OpenShiftClusterDocument
	subscriptionDocuments                    []*api.SubscriptionDocument
	billingDocuments                         []*api.BillingDocument
	asyncOperationDocuments                  []*api.AsyncOperationDocument
	portalDocuments                          []*api.PortalDocument
	gatewayDocuments                         []*api.GatewayDocument
	openShiftVersionDocuments                []*api.OpenShiftVersionDocument
	platformWorkloadIdentityRoleSetDocuments []*api.PlatformWorkloadIdentityRoleSetDocument
	validationResult                         []*api.ValidationResult
	maintenanceManifestDocuments             []*api.MaintenanceManifestDocument
}

func NewChecker() *Checker {
	return &Checker{}
}

func (f *Checker) Clear() {
	f.openshiftClusterDocuments = []*api.OpenShiftClusterDocument{}
	f.subscriptionDocuments = []*api.SubscriptionDocument{}
	f.billingDocuments = []*api.BillingDocument{}
	f.asyncOperationDocuments = []*api.AsyncOperationDocument{}
	f.portalDocuments = []*api.PortalDocument{}
	f.gatewayDocuments = []*api.GatewayDocument{}
	f.openShiftVersionDocuments = []*api.OpenShiftVersionDocument{}
	f.maintenanceManifestDocuments = []*api.MaintenanceManifestDocument{}
	f.validationResult = []*api.ValidationResult{}
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

func (f *Checker) AddOpenShiftVersionDocuments(docs ...*api.OpenShiftVersionDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.openShiftVersionDocuments = append(f.openShiftVersionDocuments, docCopy.(*api.OpenShiftVersionDocument))
	}
}

func (f *Checker) AddPlatformWorkloadIdentityRoleSetDocuments(docs ...*api.PlatformWorkloadIdentityRoleSetDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.platformWorkloadIdentityRoleSetDocuments = append(f.platformWorkloadIdentityRoleSetDocuments, docCopy.(*api.PlatformWorkloadIdentityRoleSetDocument))
	}
}

func (f *Checker) AddValidationResult(docs ...*api.ValidationResult) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.validationResult = append(f.validationResult, docCopy.(*api.ValidationResult))
	}
}

func (f *Checker) AddMaintenanceManifestDocuments(docs ...*api.MaintenanceManifestDocument) {
	for _, doc := range docs {
		docCopy, err := deepCopy(doc)
		if err != nil {
			panic(err)
		}

		f.maintenanceManifestDocuments = append(f.maintenanceManifestDocuments, docCopy.(*api.MaintenanceManifestDocument))
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

func (f *Checker) CheckOpenShiftVersions(versions *cosmosdb.FakeOpenShiftVersionDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := versions.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	sort.Slice(all.OpenShiftVersionDocuments, func(i, j int) bool { return all.OpenShiftVersionDocuments[i].ID < all.OpenShiftVersionDocuments[j].ID })

	if len(f.openShiftVersionDocuments) != 0 && len(all.OpenShiftVersionDocuments) == len(f.openShiftVersionDocuments) {
		diff := deep.Equal(all.OpenShiftVersionDocuments, f.openShiftVersionDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.OpenShiftVersionDocuments) != 0 || len(f.openShiftVersionDocuments) != 0 {
		errs = append(errs, fmt.Errorf("versions length different, %d vs %d", len(all.OpenShiftVersionDocuments), len(f.openShiftVersionDocuments)))
	}

	return errs
}

func (f *Checker) CheckPlatformWorkloadIdentityRoleSets(roleSets *cosmosdb.FakePlatformWorkloadIdentityRoleSetDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := roleSets.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	sort.Slice(all.PlatformWorkloadIdentityRoleSetDocuments, func(i, j int) bool {
		return all.PlatformWorkloadIdentityRoleSetDocuments[i].ID < all.PlatformWorkloadIdentityRoleSetDocuments[j].ID
	})

	if len(f.platformWorkloadIdentityRoleSetDocuments) != 0 && len(all.PlatformWorkloadIdentityRoleSetDocuments) == len(f.platformWorkloadIdentityRoleSetDocuments) {
		diff := deep.Equal(all.PlatformWorkloadIdentityRoleSetDocuments, f.platformWorkloadIdentityRoleSetDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.PlatformWorkloadIdentityRoleSetDocuments) != 0 || len(f.platformWorkloadIdentityRoleSetDocuments) != 0 {
		errs = append(errs, fmt.Errorf("role sets length different, %d vs %d", len(all.PlatformWorkloadIdentityRoleSetDocuments), len(f.platformWorkloadIdentityRoleSetDocuments)))
	}

	return errs
}

func (f *Checker) CheckMaintenanceManifests(client *cosmosdb.FakeMaintenanceManifestDocumentClient) (errs []error) {
	ctx := context.Background()

	all, err := client.ListAll(ctx, nil)
	if err != nil {
		return []error{err}
	}

	sort.Slice(all.MaintenanceManifestDocuments, func(i, j int) bool {
		return all.MaintenanceManifestDocuments[i].ID < all.MaintenanceManifestDocuments[j].ID
	})

	if len(f.maintenanceManifestDocuments) != 0 && len(all.MaintenanceManifestDocuments) == len(f.maintenanceManifestDocuments) {
		diff := deep.Equal(all.MaintenanceManifestDocuments, f.maintenanceManifestDocuments)
		for _, i := range diff {
			errs = append(errs, errors.New(i))
		}
	} else if len(all.MaintenanceManifestDocuments) != 0 || len(f.maintenanceManifestDocuments) != 0 {
		errs = append(errs, fmt.Errorf("document length different, %d vs %d", len(all.MaintenanceManifestDocuments), len(f.maintenanceManifestDocuments)))
	}

	return errs
}
