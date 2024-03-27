package billing

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"os"

	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/feature"
)

const (
	tenantIDMSFT = "72f988bf-86f1-41af-91ab-2d7cd011db47"
	tenantIDAME  = "33e01921-4d64-4f8c-a055-5bdaffd5e33d"
)

type Manager interface {
	Ensure(context.Context, *api.OpenShiftClusterDocument, *api.SubscriptionDocument) error
	Delete(context.Context, *api.OpenShiftClusterDocument) error
}

type manager struct {
	storageClient *azstorage.Client
	billingDB     database.Billing
	subDB         database.Subscriptions
	log           *logrus.Entry
}

func NewManager(env env.Interface, billing database.Billing, sub database.Subscriptions, log *logrus.Entry) (Manager, error) {
	storageClient, err := storageClient(env, billing, sub, log)
	if err != nil {
		return nil, err
	}

	return &manager{
		storageClient: storageClient,
		subDB:         sub,
		billingDB:     billing,
		log:           log,
	}, nil
}

func storageClient(env env.Interface, billing database.Billing, sub database.Subscriptions, log *logrus.Entry) (*azstorage.Client, error) {
	if os.Getenv("BILLING_E2E_STORAGE_ACCOUNT_ID") == "" {
		return nil, nil
	}

	r, err := azure.ParseResourceID(os.Getenv("BILLING_E2E_STORAGE_ACCOUNT_ID"))
	if err != nil {
		return nil, err
	}

	localFPAuthorizer, err := env.FPAuthorizer(env.TenantID(), env.Environment().ResourceManagerScope)
	if err != nil {
		return nil, err
	}

	e2estorage := storage.NewAccountsClient(env.Environment(), r.SubscriptionID, localFPAuthorizer)

	keys, err := e2estorage.ListKeys(context.Background(), r.ResourceGroup, r.ResourceName, "")
	if err != nil {
		return nil, err
	}

	client, err := azstorage.NewBasicClient(r.ResourceName, *(*keys.Keys)[0].Value)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (m *manager) Ensure(ctx context.Context, doc *api.OpenShiftClusterDocument, sub *api.SubscriptionDocument) error {
	billingDoc, err := m.billingDB.Create(ctx, &api.BillingDocument{
		ID:                        doc.ID,
		Key:                       doc.Key,
		ClusterResourceGroupIDKey: doc.ClusterResourceGroupIDKey,
		InfraID:                   doc.OpenShiftCluster.Properties.InfraID,
		Billing: &api.Billing{
			TenantID: sub.Subscription.Properties.TenantID,
			Location: doc.OpenShiftCluster.Location,
		},
	})
	if err, ok := err.(*cosmosdb.Error); ok &&
		err.StatusCode == http.StatusConflict {
		m.log.Print("billing record already present in DB")
		return nil
	}
	if err != nil {
		return err
	}

	if e2eErr := m.createOrUpdateE2EBlob(ctx, billingDoc); e2eErr != nil {
		m.log.Warnf("createOrUpdateE2EBlob failed: %s", e2eErr)
	}

	return nil
}

func (m *manager) Delete(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	m.log.Printf("updating billing record with deletion time")
	billingDoc, err := m.billingDB.MarkForDeletion(ctx, doc.ID)
	if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	if e2eErr := m.createOrUpdateE2EBlob(ctx, billingDoc); e2eErr != nil {
		// We are not failing the operation if we cannot write to e2e storage account, just warning
		m.log.Warnf("createOrUpdateE2EBlob failed: %s", e2eErr)
	}

	return nil
}

// isSubscriptionRegisteredForE2E returns true if the subscription has the
// "Microsoft.RedHatOpenShift/SaveAROTestConfig" feature registered
func isSubscriptionRegisteredForE2E(sub *api.SubscriptionProperties) bool {
	if sub.TenantID == tenantIDMSFT || sub.TenantID == tenantIDAME {
		return feature.IsRegisteredForFeature(sub, api.FeatureFlagSaveAROTestConfig)
	}
	return false
}
