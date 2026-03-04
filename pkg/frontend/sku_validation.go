package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	sdkcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
)

type SkuValidator interface {
	ValidateVMSku(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error
}

type skuValidator struct{}

func (s skuValidator) ValidateVMSku(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error {
	fpCredClusterTenant, err := environment.FPNewClientCertificateCredential(tenantID, nil)
	if err != nil {
		return err
	}

	armResourceSKUsClient, err := armcompute.NewResourceSKUsClient(subscriptionID, fpCredClusterTenant, environment.Environment().ArmClientOptions())
	if err != nil {
		return err
	}

	return validateVMSku(ctx, oc, armResourceSKUsClient)
}

// validateVMSku uses resourceSkusClient to ensure that the VM sizes listed in the cluster document are available for use in the target region.
func validateVMSku(ctx context.Context, oc *api.OpenShiftCluster, resourceSkusClient armcompute.ResourceSKUsClient) error {
	// Get a list of available worker SKUs, filtering by location. We initialized a new resourceSkusClient
	// so that we can determine SKU availability within target cluster subscription instead of within RP subscription.
	location := oc.Location
	filter := fmt.Sprintf("location eq %s", location)
	skus, err := resourceSkusClient.List(ctx, filter, false)
	if err != nil {
		return err
	}

	filteredSkus := computeskus.FilterVMSizes(skus, location)

	controlPlaneSKU, err := checkSKUAvailability(filteredSkus, location, "properties.masterProfile.VMSize", string(oc.Properties.MasterProfile.VMSize))
	if err != nil {
		return err
	}

	err = checkSKURestriction(controlPlaneSKU, location, "properties.masterProfile.VMSize")
	if err != nil {
		return err
	}

	if oc.Properties.MasterProfile.EncryptionAtHost == api.EncryptionAtHostEnabled {
		err = checkSKUEncryptionAtHostSupport(controlPlaneSKU, "properties.masterProfile.encryptionAtHost")
		if err != nil {
			return err
		}
	}

	workerProfiles, _ := api.GetEnrichedWorkerProfiles(oc.Properties)

	// In case there are multiple WorkerProfiles listed in the cluster document (such as post-install),
	// compare VMSize in each WorkerProfile to the resourceSkusClient call above to ensure that the sku is available in region.
	// XXX: Will this ever be called post-install?
	for i, workerprofile := range workerProfiles {
		workerProfileSku := string(workerprofile.VMSize)

		workerSKU, err := checkSKUAvailability(filteredSkus, location, fmt.Sprintf("properties.workerProfiles[%d].VMSize", i), workerProfileSku)
		if err != nil {
			return err
		}

		err = checkSKURestriction(workerSKU, location, fmt.Sprintf("properties.workerProfiles[%d].VMSize", i))
		if err != nil {
			return err
		}

		if workerprofile.EncryptionAtHost == api.EncryptionAtHostEnabled {
			err = checkSKUEncryptionAtHostSupport(workerSKU, fmt.Sprintf("properties.workerProfiles[%d].encryptionAtHost", i))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkSKUAvailability(skus map[string]*sdkcompute.ResourceSKU, location, path, vmsize string) (*sdkcompute.ResourceSKU, error) {
	// Ensure desired sku exists in target region
	sku, ok := skus[vmsize]
	if !ok {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, fmt.Sprintf("The selected SKU '%v' is unavailable in region '%v'", vmsize, location))
	}
	return sku, nil
}

func checkSKURestriction(sku *sdkcompute.ResourceSKU, location, path string) error {
	// Fail if sku is available, but restricted within the subscription. Restrictions are subscription-specific.
	// https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/error-sku-not-available
	if computeskus.IsRestricted(sku, location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, fmt.Sprintf("The selected SKU '%v' is restricted in region '%v' for selected subscription", *sku.Name, location))
	}
	return nil
}

func checkSKUEncryptionAtHostSupport(sku *sdkcompute.ResourceSKU, path string) error {
	if !computeskus.HasCapability(sku, "EncryptionAtHostSupported") {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, fmt.Sprintf("The selected SKU '%v' does not support encryption at host.", *sku.Name))
	}
	return nil
}
