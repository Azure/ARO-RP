package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
)

type SkuValidator interface {
	ValidateVMSku(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error
}

type skuValidator struct{}

func (s skuValidator) ValidateVMSku(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error {
	fpAuthorizer, err := environment.FPAuthorizer(tenantID, environment.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}
	resourceSkusClient := compute.NewResourceSkusClient(azEnv, subscriptionID, fpAuthorizer)

	return validateVMSku(ctx, oc, resourceSkusClient)
}

// validateVMSku uses resourceSkusClient to ensure that the VM sizes listed in the cluster document are available for use in the target region.
func validateVMSku(ctx context.Context, oc *api.OpenShiftCluster, resourceSkusClient compute.ResourceSkusClient) error {
	// Get a list of available worker SKUs, filtering by location. We initialized a new resourceSkusClient
	// so that we can determine SKU availability within target cluster subscription instead of within RP subscription.
	location := oc.Location
	filter := fmt.Sprintf("location eq %s", location)
	skus, err := resourceSkusClient.List(ctx, filter)
	if err != nil {
		return err
	}

	filteredSkus := computeskus.FilterVMSizes(skus, location)
	masterProfileSku := string(oc.Properties.MasterProfile.VMSize)

	err = checkSKUAvailability(filteredSkus, location, "properties.masterProfile.VMSize", masterProfileSku)
	if err != nil {
		return err
	}

	// In case there are multiple WorkerProfiles listed in the cluster document (such as post-install),
	// compare VMSize in each WorkerProfile to the resourceSkusClient call above to ensure that the sku is available in region.
	for i, workerprofile := range oc.Properties.WorkerProfiles {
		workerProfileSku := string(workerprofile.VMSize)

		err = checkSKUAvailability(filteredSkus, location, fmt.Sprintf("properties.workerProfiles[%d].VMSize", i), workerProfileSku)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkSKUAvailability(skus map[string]*mgmtcompute.ResourceSku, location, path, vmsize string) error {
	// Ensure desired sku exists in target region
	if skus[vmsize] == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, "The selected SKU '%v' is unavailable in region '%v'", vmsize, location)
	}

	// Fail if sku is available, but restricted within the subscription. Restrictions are subscription-specific.
	// https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/error-sku-not-available
	isRestricted := computeskus.IsRestricted(skus, location, vmsize)
	if isRestricted {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, "The selected SKU '%v' is restricted in region '%v' for selected subscription", vmsize, location)
	}

	return nil
}
