package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	armnetwork_sdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azurezones"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const internalLBFrontendIPName = "internal-lb-ip-v4"

var errFetchInternalLBs = errors.New("error fetching internal load balancer")
var errVMAvailability = errors.New("error determining the VM SKU availability")

func (m *manager) migrateInternalLoadBalancerZones(ctx context.Context) error {

	updateFunc := func(ctx context.Context, f database.OpenShiftClusterDocumentMutator) (*api.OpenShiftClusterDocument, error) {
		return m.db.PatchWithLease(ctx, m.doc.Key, f)
	}

	doc, err := MigrateInternalLoadBalancerZones(ctx, m.env, m.log, updateFunc, m.armLoadBalancers, m.armClusterPrivateLinkServices, m.armResourceSKUs, m.doc)
	if err != nil {
		return err
	}
	m.doc = doc
	return nil
}

func MigrateInternalLoadBalancerZones(
	ctx context.Context,
	_env env.Interface, log *logrus.Entry, updateOc database.OpenShiftClusterDocumentMutatorRunner, armLoadBalancersClient armnetwork.LoadBalancersClient, armClusterPrivateLinkServices armnetwork.PrivateLinkServicesClient, resourceSkusClient armcompute.ResourceSKUsClient, doc *api.OpenShiftClusterDocument,
) (*api.OpenShiftClusterDocument, error) {
	location := doc.OpenShiftCluster.Location
	resourceGroupName := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := doc.OpenShiftCluster.Properties.InfraID

	lb, err := GetInternalLoadBalancer(ctx, armLoadBalancersClient, doc.OpenShiftCluster.Properties)
	if err != nil {
		return doc, err
	}

	lbName := *lb.Name
	for _, config := range lb.Properties.FrontendIPConfigurations {
		if *config.Name == internalLBFrontendIPName && len(config.Zones) > 0 {
			log.Info("internal load balancer frontend IP already zone-redundant, no need to continue")
			return doc, nil
		}
	}

	filteredSkus, err := computeskus.GetVMSkusForCurrentRegion(ctx, resourceSkusClient, location)
	if err != nil {
		return doc, err
	}

	controlPlaneSKU, err := checkSKUAvailability(filteredSkus, location, "properties.masterProfile.VMSize", string(doc.OpenShiftCluster.Properties.MasterProfile.VMSize))
	if err != nil {
		return doc, errors.Join(errVMAvailability, err)
	}

	// Set RP-level options for expanded AZs
	zoneChecker := azurezones.NewManager(
		_env.FeatureIsSet(env.FeatureEnableClusterExpandedAvailabilityZones))

	controlPlaneZones := zoneChecker.FilterZones(computeskus.Zones(controlPlaneSKU))

	if len(controlPlaneZones) == 0 {
		log.Info("non-zonal control plane SKU, not adding zone-redundant frontend IP")
		return doc, nil
	}

	lbZones := []*string{}
	for _, z := range controlPlaneZones {
		lbZones = append(lbZones, pointerutils.ToPtr(z))
	}

	ilbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, infraID)
	if doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		ilbBackendPoolID = ilbBackendPoolID + "-internal-controlplane-v4"
	}

	pls, err := armClusterPrivateLinkServices.Get(ctx, resourceGroupName, infraID+"-pls", nil)
	if err != nil {
		return doc, fmt.Errorf("failure fetching PLS: %w", err)
	}

	log.Info("load balancer zonal migration: starting critical section")

	// STEP ONE: disassociate the PLS from the existing frontend IP configuration
	temporaryFIPName := fmt.Sprintf("%d-ip", _env.Now().Unix())
	temporaryFIPConfig := &armnetwork_sdk.FrontendIPConfiguration{
		Properties: &armnetwork_sdk.FrontendIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork_sdk.IPAllocationMethodDynamic),
			Subnet: &armnetwork_sdk.Subnet{
				ID: pointerutils.ToPtr(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
			},
		},
		Zones: lbZones,
		Name:  pointerutils.ToPtr(temporaryFIPName),
	}
	// firstly, create a temporary frontend IP configuration
	lb.Properties.FrontendIPConfigurations = append(lb.Properties.FrontendIPConfigurations, temporaryFIPConfig)
	err = armLoadBalancersClient.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, *lb, nil)
	if err != nil {
		log.Errorf("FAILURE IN CRITICAL SECTION: '%v'", err)
		return doc, fmt.Errorf("failure updating internal load balancer: %w", err)
	}

	// associate the temporary frontend IP with the PLS (since it always needs one)
	pls.Properties.LoadBalancerFrontendIPConfigurations = []*armnetwork_sdk.FrontendIPConfiguration{
		{
			ID: pointerutils.ToPtr(fmt.Sprintf("%s/frontendIPConfigurations/%s", *lb.ID, temporaryFIPName)),
		},
	}
	log.Infof("associating temporary frontend IP (%s) to PLS", temporaryFIPName)
	err = armClusterPrivateLinkServices.CreateOrUpdateAndWait(ctx, resourceGroupName, infraID+"-pls", pls.PrivateLinkService, nil)
	if err != nil {
		log.Errorf("FAILURE IN CRITICAL SECTION - PLS MAY NOW BE DISCONNECTED FROM LB: '%v'", err)
		return doc, fmt.Errorf("failure disassociating LB frontend IP from PLS: %w", err)
	}

	// STEP TWO: delete the existing frontend IP configuration and LB rules
	// keep the bogus config since it's in use by the PLS
	lb.Properties.FrontendIPConfigurations = []*armnetwork_sdk.FrontendIPConfiguration{temporaryFIPConfig}
	lb.Properties.LoadBalancingRules = []*armnetwork_sdk.LoadBalancingRule{}
	log.Info("removing old frontend IP")
	err = armLoadBalancersClient.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, *lb, nil)
	if err != nil {
		log.Errorf("FAILURE IN CRITICAL SECTION - API-INT RULES MAY NOW BE MISSING: '%v'", err)
		return doc, fmt.Errorf("failure updating internal load balancer: %w", err)
	}

	// STEP THREE: add a new zonal LB frontend IP with the same IP address as the old one
	frontendConfigID := fmt.Sprintf("%s/frontendIPConfigurations/%s", *lb.ID, internalLBFrontendIPName)
	newFrontendIP := &armnetwork_sdk.FrontendIPConfiguration{
		Properties: &armnetwork_sdk.FrontendIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork_sdk.IPAllocationMethodStatic),
			PrivateIPAddress:          pointerutils.ToPtr(doc.OpenShiftCluster.Properties.APIServerProfile.IntIP),
			Subnet: &armnetwork_sdk.Subnet{
				ID: pointerutils.ToPtr(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
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
		&armnetwork_sdk.LoadBalancingRule{
			Name: pointerutils.ToPtr("api-internal-v4"),
			Properties: &armnetwork_sdk.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &armnetwork_sdk.SubResource{
					ID: pointerutils.ToPtr(frontendConfigID),
				},
				BackendAddressPool: &armnetwork_sdk.SubResource{
					ID: pointerutils.ToPtr(ilbBackendPoolID),
				},
				Probe: &armnetwork_sdk.SubResource{
					ID: pointerutils.ToPtr(apiProbeID),
				},
				Protocol:             pointerutils.ToPtr(armnetwork_sdk.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(armnetwork_sdk.LoadDistributionDefault),
				FrontendPort:         pointerutils.ToPtr(int32(6443)),
				BackendPort:          pointerutils.ToPtr(int32(6443)),
				IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
				DisableOutboundSnat:  pointerutils.ToPtr(true),
			},
		},
		&armnetwork_sdk.LoadBalancingRule{
			Name: pointerutils.ToPtr("sint-v4"),
			Properties: &armnetwork_sdk.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &armnetwork_sdk.SubResource{
					ID: pointerutils.ToPtr(frontendConfigID),
				},
				BackendAddressPool: &armnetwork_sdk.SubResource{
					ID: pointerutils.ToPtr(ilbBackendPoolID),
				},
				Probe: &armnetwork_sdk.SubResource{
					ID: pointerutils.ToPtr(sintProbeID),
				},
				Protocol:             pointerutils.ToPtr(armnetwork_sdk.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(armnetwork_sdk.LoadDistributionDefault),
				FrontendPort:         pointerutils.ToPtr(int32(22623)),
				BackendPort:          pointerutils.ToPtr(int32(22623)),
				IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
			},
		},
	)

	log.Info("updating internal load balancer with zone-redundant frontend IP")
	err = armLoadBalancersClient.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, *lb, nil)
	if err != nil {
		log.Errorf("FAILURE IN CRITICAL SECTION - API-INT RULES MAY NOW BE MISSING: '%v'", err)
		return doc, fmt.Errorf("failure updating internal load balancer: %w", err)
	}

	// STEP FOUR: reassociate the frontend IP to the PLS
	log.Info("reassociating frontend IP with PLS")
	pls.Properties.LoadBalancerFrontendIPConfigurations = []*armnetwork_sdk.FrontendIPConfiguration{
		{
			ID: pointerutils.ToPtr(frontendConfigID),
		},
	}
	err = armClusterPrivateLinkServices.CreateOrUpdateAndWait(ctx, resourceGroupName, infraID+"-pls", pls.PrivateLinkService, nil)
	if err != nil {
		log.Errorf("FAILURE IN CRITICAL SECTION - PLS MAY NOW BE DISCONNECTED FROM LB: '%v'", err)
		return doc, fmt.Errorf("failure disassociating LB frontend IP from PLS: %w", err)
	}

	// STEP FIVE: remove bogus frontend IP to clean up
	log.Info("cleaning up temporary frontend IP")
	lb.Properties.FrontendIPConfigurations = []*armnetwork_sdk.FrontendIPConfiguration{newFrontendIP}
	err = armLoadBalancersClient.CreateOrUpdateAndWait(ctx, resourceGroupName, lbName, *lb, nil)
	if err != nil {
		log.Errorf("FAILURE IN CRITICAL SECTION - API-INT RULES MAY NOW BE MISSING: '%v'", err)
		return doc, fmt.Errorf("failure updating internal load balancer: %w", err)
	}

	log.Info("critical section complete, api-int migrated")

	// Update the document with the internal LB zones
	doc, err = updateOc(ctx, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.Zones = controlPlaneZones
		return nil
	})
	if err != nil {
		return doc, fmt.Errorf("failure updating cluster doc with load balancer zones: %w", err)
	}
	return doc, nil
}

func GetInternalLoadBalancer(ctx context.Context, armLoadBalancersClient armnetwork.LoadBalancersClient, ocProps api.OpenShiftClusterProperties) (*armnetwork_sdk.LoadBalancer, error) {
	infraID := ocProps.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	resourceGroup := stringutils.LastTokenByte(ocProps.ClusterProfile.ResourceGroupID, '/')

	var lbName string
	switch ocProps.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		lbName = infraID + "-internal-lb"
	case api.ArchitectureVersionV2:
		lbName = infraID + "-internal"
	default:
		return nil, fmt.Errorf("unknown architecture version %d", ocProps.ArchitectureVersion)
	}

	lb, err := armLoadBalancersClient.Get(ctx, resourceGroup, lbName, nil)
	if err != nil {
		return nil, errors.Join(errFetchInternalLBs, err)
	}
	return &lb.LoadBalancer, nil
}
