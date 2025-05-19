package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
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

func (m *manager) validateZones(ctx context.Context) error {
	location := m.doc.OpenShiftCluster.Location
	filter := fmt.Sprintf("location eq %s", location)
	skus, err := m.resourceSkus.List(ctx, filter)
	if err != nil {
		return err
	}

	filteredSkus := computeskus.FilterVMSizes(skus, location)

	controlPlaneSKU, err := checkSKUAvailability(filteredSkus, location, "properties.masterProfile.VMSize", string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize))
	if err != nil {
		return err
	}

	zoneChecker := azurezones.NewManager(false, "")
	controlPlaneZones, _, originalZones, err := zoneChecker.DetermineAvailabilityZones(controlPlaneSKU, nil)
	if err != nil {
		return err
	}

	// Update the document with the control plane and worker zones
	updatedDoc, err := m.db.PatchWithLease(ctx, m.doc.Key, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.MasterProfile.Zones = controlPlaneZones
		oscd.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.OutboundIPAvailabilityZones = originalZones
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
