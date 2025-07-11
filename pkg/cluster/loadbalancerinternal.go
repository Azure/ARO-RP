package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azurezones"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const zonalFrontendIPName = "internal-lb-ip-zonal-v4"

func (m *manager) migrateInternalLoadBalancerZones(ctx context.Context) error {
	location := m.doc.OpenShiftCluster.Location
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	var lbName string
	switch m.doc.OpenShiftCluster.Properties.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		lbName = infraID + "-internal-lb"
	case api.ArchitectureVersionV2:
		lbName = infraID + "-internal"
	default:
		return fmt.Errorf("unknown architecture version %d", m.doc.OpenShiftCluster.Properties.ArchitectureVersion)
	}

	lb, err := m.armLoadBalancers.Get(ctx, resourceGroupName, lbName, nil)
	if err != nil {
		return err
	}

	for _, config := range lb.Properties.FrontendIPConfigurations {
		if *config.Name == "internal-lb-ip-zonal-v4" {
			m.log.Info("zone-redundant frontend IP already exists, no need to continue")
			return nil
		} else if *config.Name == "internal-lb-ip-v4" && len(config.Zones) > 0 {
			m.log.Info("internal load balancer frontend IP already zone-redundant, no need to continue")
			return nil
		}
	}

	filteredSkus, err := m.getVMSKUsForCurrentRegion(ctx)
	if err != nil {
		return err
	}

	controlPlaneSKU, err := checkSKUAvailability(filteredSkus, location, "properties.masterProfile.VMSize", string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize))
	if err != nil {
		err = fmt.Errorf("error determining the VM SKU availability, skipping: %w", err)
		m.log.Error(err)
		// Don't return an error because this will stop the whole adminupdate
		return nil
	}

	// Set RP-level options for expanded AZs
	zoneChecker := azurezones.NewManager(
		m.env.FeatureIsSet(env.FeatureEnableClusterExpandedAvailabilityZones))

	controlPlaneZones := zoneChecker.FilterZones(computeskus.Zones(controlPlaneSKU))

	if len(controlPlaneZones) == 0 {
		m.log.Info("non-zonal control plane SKU, not adding zone-redundant frontend IP")
		return nil
	}

	lbZones := []*string{}
	for _, z := range controlPlaneZones {
		lbZones = append(lbZones, pointerutils.ToPtr(z))
	}

	// Update the document with the internal LB zones
	updatedDoc, err := m.db.PatchWithLease(ctx, m.doc.Key, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.Zones = controlPlaneZones
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure updating cluster doc with load balancer zones: %w", err)
	}
	m.doc = updatedDoc

	// Add a new zonal LB frontend IP
	frontendConfigID := fmt.Sprintf("%s/frontendIPConfigurations/%s", *lb.ID, zonalFrontendIPName)

	lb.Properties.FrontendIPConfigurations = append(lb.Properties.FrontendIPConfigurations,
		&armnetwork.FrontendIPConfiguration{
			Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
				PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
				Subnet: &armnetwork.Subnet{
					ID: pointerutils.ToPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
				},
			},
			Zones: lbZones,
			Name:  pointerutils.ToPtr(zonalFrontendIPName),
		})

	// Add new load balancing rules referencing the new zonal frontend IP
	ilbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, m.doc.OpenShiftCluster.Properties.InfraID)
	apiProbeID := fmt.Sprintf("%s/probes/%s", *lb.ID, "api-internal-probe")
	sintProbeID := fmt.Sprintf("%s/probes/%s", *lb.ID, "sint-probe")

	lb.Properties.LoadBalancingRules = append(lb.Properties.LoadBalancingRules,
		&armnetwork.LoadBalancingRule{
			Name: pointerutils.ToPtr("api-internal-v4-zonal"),
			Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(frontendConfigID),
				},
				BackendAddressPool: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(ilbBackendPoolID),
				},
				Probe: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(apiProbeID),
				},
				Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
				FrontendPort:         pointerutils.ToPtr(int32(6443)),
				BackendPort:          pointerutils.ToPtr(int32(6443)),
				IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
				DisableOutboundSnat:  pointerutils.ToPtr(true),
			},
		},
		&armnetwork.LoadBalancingRule{
			Name: pointerutils.ToPtr("sint-v4-zonal"),
			Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(frontendConfigID),
				},
				BackendAddressPool: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(ilbBackendPoolID),
				},
				Probe: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(sintProbeID),
				},
				Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
				FrontendPort:         pointerutils.ToPtr(int32(22623)),
				BackendPort:          pointerutils.ToPtr(int32(22623)),
				IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
			},
		},
	)

	m.log.Info("updating internal load balancer with zone-redundant frontend IP")
	err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, lb.LoadBalancer, nil)
	if err != nil {
		return fmt.Errorf("failure updating internal load balancer: %w", err)
	}
	return nil
}
