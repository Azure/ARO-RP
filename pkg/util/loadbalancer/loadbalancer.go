package loadbalancer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
)

func RemoveFrontendIPConfiguration(lb *armnetwork.LoadBalancer, resourceID string) error {
	newFrontendIPConfig := make([]*armnetwork.FrontendIPConfiguration, 0, len(lb.Properties.FrontendIPConfigurations))
	for _, fipConfig := range lb.Properties.FrontendIPConfigurations {
		if fipConfig.ID != nil && strings.EqualFold(*fipConfig.ID, resourceID) {
			if isFrontendIPConfigReferenced(fipConfig) {
				return fmt.Errorf("frontend IP Configuration %s has external references, remove the external references prior to removing the frontend IP configuration", resourceID)
			}
			continue
		}
		newFrontendIPConfig = append(newFrontendIPConfig, fipConfig)
	}
	lb.Properties.FrontendIPConfigurations = newFrontendIPConfig
	return nil
}

func isFrontendIPConfigReferenced(fipConfig *armnetwork.FrontendIPConfiguration) bool {
	return fipConfig.Properties.LoadBalancingRules != nil || fipConfig.Properties.InboundNatPools != nil || fipConfig.Properties.InboundNatRules != nil || fipConfig.Properties.OutboundRules != nil
}

func RemoveHealthProbe(lb *armnetwork.LoadBalancer, resourceID string) error {
	newProbes := make([]*armnetwork.Probe, 0)

	// Iterate over probes, build a list without the targeted resourceID, if possible
	for _, probe := range lb.Properties.Probes {
		if probe.ID != nil && strings.EqualFold(*probe.ID, resourceID) {
			if probe.Properties.LoadBalancingRules != nil {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("Load balancer health probe %s is used by load balancing rules, remove the referencing load balancing rules before removing the health probe", resourceID))
			}
			continue
		}
		newProbes = append(newProbes, probe)
	}
	lb.Properties.Probes = newProbes

	//return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeInvalidParameter, "", "No health probes found for this loadbalancer")

	return nil
}
