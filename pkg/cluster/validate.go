package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	allowExpandedAZs := m.env.FeatureIsSet(env.FeatureEnableClusterExpandedAvailabilityZones)
	zoneChecker := azurezones.NewManager(allowExpandedAZs)

	_, _, originalZones, err := zoneChecker.DetermineAvailabilityZones(controlPlaneSKU, workerSKU)
	if err != nil {
		var zoneErr *azurezones.ZoneError
		if errors.As(err, &zoneErr) {
			return zoneErrorAsCloudError(zoneErr, location, allowExpandedAZs)
		}
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
func checkSKUAvailability(skus map[string]*armcompute.ResourceSKU, location, path, vmsize string) (*armcompute.ResourceSKU, error) {
	// Ensure desired sku exists in target region
	sku, ok := skus[vmsize]
	if !ok {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, fmt.Sprintf("The selected SKU '%v' is unavailable in region '%v'", vmsize, location))
	}
	return sku, nil
}

// zoneErrorAsCloudError turns a failed availability zone check into a
// customer-facing CloudError. The wording lives here rather than in azurezones
// so that package stays pure zone logic.
func zoneErrorAsCloudError(zoneErr *azurezones.ZoneError, location string, allowExpandedAvailabilityZones bool) error {
	var target string
	switch zoneErr.Profile {
	case azurezones.VMRoleMaster:
		target = "properties.masterProfile.VMSize"
	case azurezones.VMRoleWorker:
		target = "properties.workerProfiles[0].VMSize"
	}

	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, target,
		zoneErrorMessage(zoneErr, location, allowExpandedAvailabilityZones))
}

// zoneRoleNames maps a VM role to the wording used in customer-facing
// messages, keeping all display text in this package.
var zoneRoleNames = map[azurezones.VMRole]string{
	azurezones.VMRoleMaster: "control plane",
	azurezones.VMRoleWorker: "worker",
}

// otherRole returns the role paired with the given one.
func otherRole(role azurezones.VMRole) azurezones.VMRole {
	if role == azurezones.VMRoleMaster {
		return azurezones.VMRoleWorker
	}
	return azurezones.VMRoleMaster
}

func zoneErrorMessage(zoneErr *azurezones.ZoneError, location string, allowExpandedAvailabilityZones bool) string {
	role := zoneRoleNames[zoneErr.Profile]
	supportedZones := strings.Join(azurezones.SupportedZones, ", ")

	if errors.Is(zoneErr.Err, azurezones.ErrMixedZonal) {
		otherRole := zoneRoleNames[otherRole(zoneErr.Profile)]
		return fmt.Sprintf("The %s SKU '%s' is not available in availability zones in region '%s', but the %s SKU '%s' is. Zonal and non-zonal SKUs cannot be mixed in a cluster. Please select SKUs that are either both zonal or both non-zonal in this region.",
			role, zoneErr.SKUName, location, otherRole, zoneErr.OtherSKUName)
	}

	// With expanded AZs enabled the supported zone count is the zone count
	// Azure reports, so it can be compared against the requirement directly.
	if allowExpandedAvailabilityZones {
		return fmt.Sprintf("The selected %s SKU '%s' is available in only %d of the %d required availability zones in region '%s'. Please select a SKU that is available in at least %d availability zones in this region.",
			role, zoneErr.SKUName, zoneErr.SupportedZones, zoneErr.RequiredZones, location, zoneErr.RequiredZones)
	}

	if zoneErr.SupportedZones == 0 {
		return fmt.Sprintf("The selected %s SKU '%s' is available in %d availability zone(s) in region '%s', but none of them are zones %s, which ARO requires for zonal cluster creation.",
			role, zoneErr.SKUName, zoneErr.SKUZones, location, supportedZones)
	}

	return fmt.Sprintf("The selected %s SKU '%s' is available in only %d of the %d required availability zones in region '%s' (zones %s). Please select a SKU that is available in zones %s in this region.",
		role, zoneErr.SKUName, zoneErr.SupportedZones, zoneErr.RequiredZones, location, supportedZones, supportedZones)
}
