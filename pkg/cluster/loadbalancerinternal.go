package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azurezones"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const internalLBFrontendIPName = "internal-lb-ip-v4"

var errFetchInternalLBs = errors.New("error fetching internal load balancer")
var errVMAvailability = errors.New("error determining the VM SKU availability")

func (m *manager) migrateInternalLoadBalancerZones(ctx context.Context) error {
	location := m.doc.OpenShiftCluster.Location
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	lb, err := m.getInternalLoadBalancer(ctx)
	if err != nil {
		return err
	}

	lbName := *lb.Name
	for _, config := range lb.Properties.FrontendIPConfigurations {
		if *config.Name == internalLBFrontendIPName && len(config.Zones) > 0 {
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
		return errors.Join(errVMAvailability, err)
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

	ilbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, infraID)
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		ilbBackendPoolID = ilbBackendPoolID + "-internal-controlplane-v4"
	}

	pls, err := m.armClusterPrivateLinkServices.Get(ctx, resourceGroupName, infraID+"-pls", nil)
	if err != nil {
		return fmt.Errorf("failure fetching PLS: %w", err)
	}

	m.log.Info("load balancer zonal migration: starting critical section")

	// STEP ONE: disassociate the PLS from the existing frontend IP configuration
	temporaryFIPName := fmt.Sprintf("%d-ip", m.now().Unix())
	temporaryFIPConfig := &armnetwork.FrontendIPConfiguration{
		Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
			Subnet: &armnetwork.Subnet{
				ID: pointerutils.ToPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
			},
		},
		Zones: lbZones,
		Name:  pointerutils.ToPtr(temporaryFIPName),
	}
	// firstly, create a temporary frontend IP configuration
	lb.Properties.FrontendIPConfigurations = append(lb.Properties.FrontendIPConfigurations, temporaryFIPConfig)
	err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, *lb, nil)
	if err != nil {
		m.log.Errorf("FAILURE IN CRITICAL SECTION: '%v'", err)
		return fmt.Errorf("failure updating internal load balancer: %w", err)
	}

	// associate the temporary frontend IP with the PLS (since it always needs one)
	pls.Properties.LoadBalancerFrontendIPConfigurations = []*armnetwork.FrontendIPConfiguration{
		{
			ID: pointerutils.ToPtr(fmt.Sprintf("%s/frontendIPConfigurations/%s", *lb.ID, temporaryFIPName)),
		},
	}
	m.log.Infof("associating temporary frontend IP (%s) to PLS", temporaryFIPName)
	err = m.armClusterPrivateLinkServices.CreateOrUpdateAndWait(ctx, resourceGroupName, infraID+"-pls", pls.PrivateLinkService, nil)
	if err != nil {
		m.log.Errorf("FAILURE IN CRITICAL SECTION - PLS MAY NOW BE DISCONNECTED FROM LB: '%v'", err)
		return fmt.Errorf("failure disassociating LB frontend IP from PLS: %w", err)
	}

	// STEP TWO: delete the existing frontend IP configuration and LB rules
	// keep the bogus config since it's in use by the PLS
	lb.Properties.FrontendIPConfigurations = []*armnetwork.FrontendIPConfiguration{temporaryFIPConfig}
	lb.Properties.LoadBalancingRules = []*armnetwork.LoadBalancingRule{}
	m.log.Info("removing old frontend IP")
	err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, *lb, nil)
	if err != nil {
		m.log.Errorf("FAILURE IN CRITICAL SECTION - API-INT RULES MAY NOW BE MISSING: '%v'", err)
		return fmt.Errorf("failure updating internal load balancer: %w", err)
	}

	// STEP THREE: add a new zonal LB frontend IP with the same IP address as the old one
	frontendConfigID := fmt.Sprintf("%s/frontendIPConfigurations/%s", *lb.ID, internalLBFrontendIPName)
	newFrontendIP := &armnetwork.FrontendIPConfiguration{
		Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodStatic),
			PrivateIPAddress:          pointerutils.ToPtr(m.doc.OpenShiftCluster.Properties.APIServerProfile.IntIP),
			Subnet: &armnetwork.Subnet{
				ID: pointerutils.ToPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
			},
		},
		Zones: lbZones,
		Name:  pointerutils.ToPtr(internalLBFrontendIPName),
	}

	lb.Properties.FrontendIPConfigurations = append(lb.Properties.FrontendIPConfigurations, newFrontendIP)

	// Add new load balancing rules referencing the new zonal frontend IP
	apiProbeID := fmt.Sprintf("%s/probes/%s", *lb.ID, "api-internal-probe")
	sintProbeID := fmt.Sprintf("%s/probes/%s", *lb.ID, "sint-probe")

	lb.Properties.LoadBalancingRules = append(lb.Properties.LoadBalancingRules,
		&armnetwork.LoadBalancingRule{
			Name: pointerutils.ToPtr("api-internal-v4"),
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
			Name: pointerutils.ToPtr("sint-v4"),
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
	err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, *lb, nil)
	if err != nil {
		m.log.Errorf("FAILURE IN CRITICAL SECTION - API-INT RULES MAY NOW BE MISSING: '%v'", err)
		return fmt.Errorf("failure updating internal load balancer: %w", err)
	}

	// STEP FOUR: reassociate the frontend IP to the PLS
	m.log.Info("reassociating frontend IP with PLS")
	pls.Properties.LoadBalancerFrontendIPConfigurations = []*armnetwork.FrontendIPConfiguration{
		{
			ID: pointerutils.ToPtr(frontendConfigID),
		},
	}
	err = m.armClusterPrivateLinkServices.CreateOrUpdateAndWait(ctx, resourceGroupName, infraID+"-pls", pls.PrivateLinkService, nil)
	if err != nil {
		m.log.Errorf("FAILURE IN CRITICAL SECTION - PLS MAY NOW BE DISCONNECTED FROM LB: '%v'", err)
		return fmt.Errorf("failure disassociating LB frontend IP from PLS: %w", err)
	}

	// STEP FIVE: remove bogus frontend IP to clean up
	m.log.Info("cleaning up temporary frontend IP")
	lb.Properties.FrontendIPConfigurations = []*armnetwork.FrontendIPConfiguration{newFrontendIP}
	err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, *lb, nil)
	if err != nil {
		m.log.Errorf("FAILURE IN CRITICAL SECTION - API-INT RULES MAY NOW BE MISSING: '%v'", err)
		return fmt.Errorf("failure updating internal load balancer: %w", err)
	}

	m.log.Info("critical section complete, api-int migrated")

	// Update the document with the internal LB zones
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.Zones = controlPlaneZones
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure updating cluster doc with load balancer zones: %w", err)
	}
	return nil
}

func (m *manager) getInternalLoadBalancer(ctx context.Context) (*armnetwork.LoadBalancer, error) {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	var lbName string
	switch m.doc.OpenShiftCluster.Properties.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		lbName = infraID + "-internal-lb"
	case api.ArchitectureVersionV2:
		lbName = infraID + "-internal"
	default:
		return nil, fmt.Errorf("unknown architecture version %d", m.doc.OpenShiftCluster.Properties.ArchitectureVersion)
	}

	lb, err := m.armLoadBalancers.Get(ctx, resourceGroup, lbName, nil)
	if err != nil {
		return nil, errors.Join(errFetchInternalLBs, err)
	}
	return &lb.LoadBalancer, nil
}
