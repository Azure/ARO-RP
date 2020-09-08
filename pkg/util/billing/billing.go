package billing

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

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

	prodE2EStorageAccountSubID  = "0923c7de-9fca-4d9e-baf3-131d0c5b2ea4"
	prodE2EStorageAccountRGName = "global"
	prodE2EStorageAccountName   = "arov4e2e"
	intE2EStorageAccountSubID   = "0cc1cafa-578f-4fa5-8d6b-ddfd8d82e6ea"
	intE2EStorageAccountRGName  = "global-infra"
	intE2EStorageAccountName    = "arov4e2eint"

	// featureSaveAROTestConfig is the feature in the subscription that is used
	// to indicate if we need to save ARO cluster config into the E2E
	// StorageAccount
	featureSaveAROTestConfig = "Microsoft.RedHatOpenShift/SaveAROTestConfig"
)

type Manager interface {
	Ensure(context.Context, *api.OpenShiftClusterDocument) error
	Delete(context.Context, *api.OpenShiftClusterDocument) error
}

type manager struct {
	env             env.Interface
	storageClient   *azstorage.Client
	dbbilling       database.Billing
	dbsubscriptions database.Subscriptions
	log             *logrus.Entry
}

func NewManager(_env env.Interface, fp env.FPAuthorizer, dbbilling database.Billing, dbsubscriptions database.Subscriptions, log *logrus.Entry) (Manager, error) {
	storageClient, err := getStorageClient(_env, fp)
	if err != nil {
		return nil, err
	}

	return &manager{
		env:             _env,
		storageClient:   storageClient,
		dbsubscriptions: dbsubscriptions,
		dbbilling:       dbbilling,
		log:             log,
	}, nil
}

func getStorageClient(_env env.Interface, fp env.FPAuthorizer) (*azstorage.Client, error) {
	var e2eStorageAccountSubID, e2eStorageAccountRGName, e2eStorageAccountName string

	switch _env.Type() {
	case env.Dev:
		return nil, nil // allowed: createOrUpdateE2EBlob does nothing in dev mode
	case env.Int:
		e2eStorageAccountSubID = intE2EStorageAccountSubID
		e2eStorageAccountRGName = intE2EStorageAccountRGName
		e2eStorageAccountName = intE2EStorageAccountName
	default:
		e2eStorageAccountSubID = prodE2EStorageAccountSubID
		e2eStorageAccountRGName = prodE2EStorageAccountRGName
		e2eStorageAccountName = prodE2EStorageAccountName
	}

	localFPAuthorizer, err := fp.FPAuthorizer(_env.TenantID(), azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	e2estorage := storage.NewAccountsClient(e2eStorageAccountSubID, localFPAuthorizer)

	keys, err := e2estorage.ListKeys(context.Background(), e2eStorageAccountRGName, e2eStorageAccountName, "")
	if err != nil {
		return nil, err
	}
	key := *(*keys.Keys)[0].Value

	client, err := azstorage.NewBasicClient(e2eStorageAccountName, key)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (m *manager) Ensure(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	billingDoc, err := m.dbbilling.Create(ctx, &api.BillingDocument{
		ID:                        doc.ID,
		Key:                       doc.Key,
		ClusterResourceGroupIDKey: doc.ClusterResourceGroupIDKey,
		InfraID:                   doc.OpenShiftCluster.Properties.InfraID,
		Billing: &api.Billing{
			TenantID: doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
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
	billingDoc, err := m.dbbilling.MarkForDeletion(ctx, doc.ID)
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
		return feature.IsRegisteredForFeature(sub, featureSaveAROTestConfig)
	}
	return false
}

// createOrUpdateE2Eblob create a copy of the billing document in the e2e
// storage account. This is used later on by the billing e2e
func (m *manager) createOrUpdateE2EBlob(ctx context.Context, doc *api.BillingDocument) error {
	//skip updating the storage account if this is a dev scenario
	if m.env.Type() == env.Dev {
		return nil
	}

	// Validate if E2E Feature is registered
	resource, err := azure.ParseResourceID(doc.Key)
	if err != nil {
		return err
	}

	subscriptionDoc, err := m.dbsubscriptions.Get(ctx, resource.SubscriptionID)
	if err != nil {
		return err
	}

	if !isSubscriptionRegisteredForE2E(subscriptionDoc.Subscription.Properties) {
		return nil
	}

	blobclient := m.storageClient.GetBlobService()

	containerName := strings.ToLower("bill-" + doc.Billing.Location + "-" + resource.ResourceGroup + "-" + resource.ResourceName)
	if len(containerName) > 63 {
		containerName = containerName[:63]
	}

	containerRef := blobclient.GetContainerReference(containerName)
	_, err = containerRef.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	blobRef := containerRef.GetBlobReference("billingentity")
	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	return blobRef.CreateBlockBlobFromReader(bytes.NewReader(b), nil)
}
