package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

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

func (m *manager) getVMSKUsForCurrentRegion(ctx context.Context) (map[string]*mgmtcompute.ResourceSku, error) {
	location := m.doc.OpenShiftCluster.Location
	filter := fmt.Sprintf("location eq %s", location)
	skus, err := m.resourceSkus.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failure listing resource SKUs: %w", err)
	}

	return computeskus.FilterVMSizes(skus, location), nil
}

// validateZones validates the SKU availability and zones of the cluster being
// created. This function is only to be called during cluster bootstrap!
func (m *manager) validateZones(ctx context.Context) error {
	location := m.doc.OpenShiftCluster.Location
	filteredSkus, err := m.getVMSKUsForCurrentRegion(ctx)
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

	_, _, originalZones, err := zoneChecker.DetermineAvailabilityZones(controlPlaneSKU, workerSKU)
	if err != nil {
		return err
	}

	// Update the document with configured zones
	updatedDoc, err := m.db.PatchWithLease(ctx, m.doc.Key, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.Zones = originalZones
		return nil
	})
	m.doc = updatedDoc

	return err
}

// see pkg/frontend/sku_validation.go
func checkSKUAvailability(skus map[string]*mgmtcompute.ResourceSku, location, path, vmsize string) (*mgmtcompute.ResourceSku, error) {
	// Ensure desired sku exists in target region
	sku, ok := skus[vmsize]
	if !ok {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, fmt.Sprintf("The selected SKU '%v' is unavailable in region '%v'", vmsize, location))
	}
	return sku, nil
}
