package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const snatPortsPerIP = 63992

func (dv *dynamic) ValidateLoadBalancerProfile(ctx context.Context, oc *api.OpenShiftCluster) error {
	dv.log.Print("ValidateLoadBalancerProfile")

	if oc.Properties.NetworkProfile.OutboundType == api.OutboundTypeUserDefinedRouting {
		return nil
	}

	if oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		err := dv.validatePublicIPQuota(ctx, oc)
		if err != nil {
			return err
		}
	}

	err := dv.validateOBRuleV4FrontendPorts(ctx, oc)
	if err != nil {
		return err
	}

	return nil
}

// public IP quota is also checked on the frontend, but only during cluster creation
func (dv *dynamic) validatePublicIPQuota(ctx context.Context, oc *api.OpenShiftCluster) error {
	dv.log.Print("validatePublicIPQuota")

	requestedIPs := oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count

	if oc.Properties.ProvisioningState == api.ProvisioningStateCreating && oc.Properties.IngressProfiles[0].Visibility == api.VisibilityPublic {
		requestedIPs += 1
	} else if oc.Properties.ProvisioningState == api.ProvisioningStateUpdating {
		currentIPs := len(oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs)
		if requestedIPs > currentIPs {
			requestedIPs = requestedIPs - currentIPs
		}
	}

	netUsages, err := dv.spNetworkUsage.List(ctx, oc.Location, nil)
	if err != nil {
		return err
	}

	for _, netUsage := range netUsages {
		if *netUsage.Name.Value == "PublicIPAddresses" {
			if int64(requestedIPs) > (*netUsage.Limit - *netUsage.CurrentValue) {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceQuotaExceeded, "properties.networkProfile.loadBalancerProfile.ManagedOutboundIPs.Count", "Resource quota of %s exceeded. Maximum allowed: %d, Current in use: %d, Additional requested: %d.", *netUsage.Name.Value, *netUsage.Limit, *netUsage.CurrentValue, requestedIPs)
			}
		}
	}
	return nil
}

func (dv *dynamic) validateOBRuleV4FrontendPorts(ctx context.Context, oc *api.OpenShiftCluster) error {
	dv.log.Print("validateOBRuleV4FrontendPorts")
	if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
		return nil
	}

	rgName := stringutils.LastTokenByte(oc.Properties.ClusterProfile.ResourceGroupID, '/')
	loadBalancerName := oc.Properties.InfraID
	backendAddressPoolName := oc.Properties.InfraID

	backendPools, err := dv.loadBalancerBackendAddressPoolsClient.Get(ctx, rgName, loadBalancerName, backendAddressPoolName)
	if err != nil {
		return err
	}

	totalBackendInstances := len(*backendPools.BackendAddressPoolPropertiesFormat.BackendIPConfigurations)
	// TODO: update once allocatedOutboundPorts is implemented
	allocatedOutboundPorts := 1024
	var desiredNumIPs int
	// TODO: add OutboundIPs and OutboundIPPrefixes
	if oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		desiredNumIPs = oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count
	}
	totalSNATPorts := desiredNumIPs * snatPortsPerIP
	maxBackendInstances := totalSNATPorts / allocatedOutboundPorts

	if totalBackendInstances > maxBackendInstances {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"properties.networkProfile.loadBalancerProfile",
			"Insufficient frontend ports to support the backend instance count.  Total frontend ports: %d, Required frontend ports: %d, Total backend instances: %d", totalSNATPorts, allocatedOutboundPorts*totalBackendInstances, totalBackendInstances,
		)
	}

	return nil
}
