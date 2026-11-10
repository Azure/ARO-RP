package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/ARO-RP/pkg/util/azurezones"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	"github.com/Azure/ARO-RP/pkg/validate"
)

func (m *manager) validateResources(ctx context.Context) error {
	var clusterMSICredential azcore.TokenCredential
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		clusterMSICredential = m.userAssignedIdentities.GetClusterMSICredential()
	}
	return validate.NewOpenShiftClusterDynamicValidator(
		m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.fpAuthorizer, m.armRoleDefinitions, m.clusterMsiFederatedIdentityCredentials, m.platformWorkloadIdentities, m.platformWorkloadIdentityRolesByVersion, clusterMSICredential,
	).Dynamic(ctx)
}

// validateZones validates the SKU availability and zones of the cluster being
// created. This function is only to be called during cluster bootstrap!
func (m *manager) validateZones(ctx context.Context) error {
	location := m.doc.OpenShiftCluster.Location

	filteredSkus, err := computeskus.SelectVMSkusInCurrentRegion(ctx, m.armResourceSKUs, location, []string{
		string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize),
		string(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].VMSize),
	})
	if err != nil {
		return err
	}

	controlPlaneSKU, err := checkSKUAvailability(filteredSkus, location, "properties.masterProfile.VMSize", string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize))
	if err != nil {
		return err
	}

	workerSKU, err := checkSKUAvailability(filteredSkus, location, "properties.workerProfiles[0].VMSize", string(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].VMSize))
	if err != nil {
		return err
	}

	// Set RP-level options for expanded AZs
	zoneChecker := azurezones.NewManager(
		m.env.FeatureIsSet(env.FeatureEnableClusterExpandedAvailabilityZones))

	if m.doc.OpenShiftCluster.Properties.Zones == nil {
		// If the document does not have the list of AZs (nil list) then
		// determine it the old way (by looking at the SKUs)
		_, _, originalZones, err := zoneChecker.DetermineAvailabilityZones(controlPlaneSKU, workerSKU)
		if err != nil {
			return err
		}
		updatedDoc, err := m.db.PatchWithLease(ctx, m.doc.Key, func(oscd *api.OpenShiftClusterDocument) error {
			oscd.OpenShiftCluster.Properties.Zones = originalZones
			return nil
		})
		m.doc = updatedDoc
	} else if len(m.doc.OpenShiftCluster.Properties.Zones) == 0 {
		// We haven't requested availability zones, so don't check for them (zero-length but allocated list)
		return nil
	} else {
		ok := zoneChecker.DesiredZonesAvailable(controlPlaneSKU, workerSKU, m.doc.OpenShiftCluster.Properties.Zones)
		if !ok {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.Zones", fmt.Sprintf("The selected zones (%v) are not available across both SKUs", m.doc.OpenShiftCluster.Properties.Zones))
		}
	}
	return nil
}

// see pkg/frontend/sku_validation.go
func checkSKUAvailability(skus map[string]*armcompute.ResourceSKU, location, path, vmsize string) (*armcompute.ResourceSKU, error) {
	// Ensure desired sku exists in target region
	sku, ok := skus[vmsize]
	if !ok {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, fmt.Sprintf("The selected SKU '%v' is unavailable in region '%v'", vmsize, location))
	}
	return sku, nil
}
