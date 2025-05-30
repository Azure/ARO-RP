package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azurezones"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/go-autorest/autorest/to"
)

const zonalFrontendIPName = "internal-lb-ip-zonal-v4"

func (m *manager) fixInternalLoadBalancerZones(ctx context.Context) error {
	// todo: architecture v1
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion != api.ArchitectureVersionV2 {
		m.log.Info("skipping internal load balancer zonality fixing because cluster is not architecture v2")
		return nil
	}
	location := m.doc.OpenShiftCluster.Location

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	lb, err := m.armLoadBalancers.Get(ctx, resourceGroupName, infraID+"-internal", nil)
	if err != nil {
		return err
	}

	for _, config := range lb.Properties.FrontendIPConfigurations {
		if *config.Name == "internal-lb-ip-zonal-v4" {
			m.log.Info("zonal frontend IP already exists, no need to continue")
			return nil
		} else if *config.Name == "internal-lb-ip-v4" && len(config.Zones) > 0 {
			m.log.Info("frontend IP already created with zones, no need to continue")
			return nil
		}
	}

	filteredSkus, err := m.getVMSKUsForCurrentRegion(ctx)
	if err != nil {
		return err
	}

	controlPlaneSKU, err := checkSKUAvailability(filteredSkus, location, "properties.masterProfile.VMSize", string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize))
	if err != nil {
		return err
	}

	// Set RP-level options for expanded AZs
	zoneChecker := azurezones.NewManager(
		m.env.FeatureIsSet(env.FeatureEnableClusterExpandedAvailabilityZones),
		false, "")

	controlPlaneZones := zoneChecker.FilterZones(computeskus.Zones(controlPlaneSKU))

	if len(controlPlaneZones) == 0 {
		m.log.Info("Non-zonal control plane SKU, not adding zonal frontend IP")
		return nil
	}

	lbZones := []*string{}
	for _, z := range controlPlaneZones {
		lbZones = append(lbZones, to.StringPtr(z))
	}

	// Update the document with the internal LB zones
	updatedDoc, err := m.db.PatchWithLease(ctx, m.doc.Key, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.NetworkProfile.InternalLoadBalancerZones = controlPlaneZones
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
				PrivateIPAllocationMethod: pointerutils.ToPtr(sdknetwork.IPAllocationMethodDynamic),
				Subnet: &armnetwork.Subnet{
					ID: to.StringPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
				},
			},
			Zones: lbZones,
			Name:  to.StringPtr(zonalFrontendIPName),
		})

	// Add new load balancing rules referencing the new zonal frontend IP
	ilbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, m.doc.OpenShiftCluster.Properties.InfraID)
	apiProbeID := fmt.Sprintf("%s/probes/%s", *lb.ID, "api-internal-probe")
	sintProbeID := fmt.Sprintf("%s/probes/%s", *lb.ID, "sint-probe")

	lb.Properties.LoadBalancingRules = append(lb.Properties.LoadBalancingRules,
		&sdknetwork.LoadBalancingRule{
			Name: to.StringPtr("api-internal-v4-zonal"),
			Properties: &sdknetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &sdknetwork.SubResource{
					ID: to.StringPtr(frontendConfigID),
				},
				BackendAddressPool: &sdknetwork.SubResource{
					ID: to.StringPtr(ilbBackendPoolID),
				},
				Probe: &sdknetwork.SubResource{
					ID: to.StringPtr(apiProbeID),
				},
				Protocol:             pointerutils.ToPtr(sdknetwork.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(sdknetwork.LoadDistributionDefault),
				FrontendPort:         to.Int32Ptr(6443),
				BackendPort:          to.Int32Ptr(6443),
				IdleTimeoutInMinutes: to.Int32Ptr(30),
				DisableOutboundSnat:  to.BoolPtr(true),
			},
		},
		&sdknetwork.LoadBalancingRule{
			Name: to.StringPtr("sint-v4-zonal"),
			Properties: &sdknetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &sdknetwork.SubResource{
					ID: to.StringPtr(frontendConfigID),
				},
				BackendAddressPool: &sdknetwork.SubResource{
					ID: to.StringPtr(ilbBackendPoolID),
				},
				Probe: &sdknetwork.SubResource{
					ID: to.StringPtr(sintProbeID),
				},
				Protocol:             pointerutils.ToPtr(sdknetwork.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(sdknetwork.LoadDistributionDefault),
				FrontendPort:         to.Int32Ptr(22623),
				BackendPort:          to.Int32Ptr(22623),
				IdleTimeoutInMinutes: to.Int32Ptr(30),
			},
		},
	)

	err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroupName, infraID+"-internal", lb.LoadBalancer, nil)
	if err != nil {
		return fmt.Errorf("failure updating internal load balancer: %w", err)
	}
	return nil

}
