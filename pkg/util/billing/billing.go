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
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/feature"
)

const (
	tenantIDMSFT = "72f988bf-86f1-41af-91ab-2d7cd011db47"
	tenantIDAME  = "33e01921-4d64-4f8c-a055-5bdaffd5e33d"

	// featureSaveAROTestConfig is the feature in the subscription that is used
	// to indicate if we need to save ARO cluster config into the E2E
	// StorageAccount
	featureSaveAROTestConfig = "Microsoft.RedHatOpenShift/SaveAROTestConfig"

	prodE2ESubscriptionID     = "0923c7de-9fca-4d9e-baf3-131d0c5b2ea4"
	prodE2EResourceGroupName  = "global"
	prodE2EStorageAccountName = "arov4e2e"

	intE2ESubscriptionID     = "0cc1cafa-578f-4fa5-8d6b-ddfd8d82e6ea"
	intE2EResourceGroupName  = "global-infra"
	intE2EStorageAccountName = "arov4e2eint"
)

type Manager interface {
	Ensure(context.Context, *api.OpenShiftClusterDocument) error
	Delete(context.Context, *api.OpenShiftClusterDocument) error
}

type manager struct {
	env           env.Interface
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
		env:           env,
		storageClient: storageClient,
		subDB:         sub,
		billingDB:     billing,
		log:           log,
	}, nil
}

func storageClient(env env.Interface, billing database.Billing, sub database.Subscriptions, log *logrus.Entry) (*azstorage.Client, error) {
	subscriptionID := prodE2ESubscriptionID
	resourceGroupName := prodE2EResourceGroupName
	storageAccountName := prodE2EStorageAccountName

	switch env.DeploymentMode() {
	case deployment.Development:
		return nil, nil

	case deployment.Integration:
		subscriptionID = intE2ESubscriptionID
		resourceGroupName = intE2EResourceGroupName
		storageAccountName = intE2EStorageAccountName
	}

	localFPAuthorizer, err := env.FPAuthorizer(env.TenantID(), env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	e2estorage := storage.NewAccountsClient(env.Environment(), subscriptionID, localFPAuthorizer)

	keys, err := e2estorage.ListKeys(context.Background(), resourceGroupName, storageAccountName, "")
	if err != nil {
		return nil, err
	}

	client, err := azstorage.NewBasicClient(storageAccountName, *(*keys.Keys)[0].Value)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (m *manager) Ensure(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	billingDoc, err := m.billingDB.Create(ctx, &api.BillingDocument{
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
		return feature.IsRegisteredForFeature(sub, featureSaveAROTestConfig)
	}
	return false
}

// createOrUpdateE2Eblob create a copy of the billing document in the e2e
// storage account. This is used later on by the billing e2e
func (m *manager) createOrUpdateE2EBlob(ctx context.Context, doc *api.BillingDocument) error {
	//skip updating the storage account if this is a dev scenario
	if m.env.DeploymentMode() == deployment.Development {
		return nil
	}

	// Validate if E2E Feature is registered
	resource, err := azure.ParseResourceID(doc.Key)
	if err != nil {
		return err
	}

	subscriptionDoc, err := m.subDB.Get(ctx, resource.SubscriptionID)
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

	// The following is added to get rid of the '-' at the end in order to avoid an invalid container name.
	containerName = strings.TrimSuffix(containerName, "-")

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
