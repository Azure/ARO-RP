package loadbalancer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

func RemoveFrontendIPConfiguration(lb *mgmtnetwork.LoadBalancer, resourceID string) error {
	newFrontendIPConfig := make([]mgmtnetwork.FrontendIPConfiguration, 0, len(*lb.FrontendIPConfigurations))
	for _, fipConfig := range *lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations {
		if strings.EqualFold(*fipConfig.ID, resourceID) {
			if isFrontendIPConfigReferenced(fipConfig) {
				return fmt.Errorf("frontend IP Configuration %s has external references, remove the external references prior to removing the frontend IP configuration", resourceID)
			}
			continue
		}
		newFrontendIPConfig = append(newFrontendIPConfig, fipConfig)
	}
	lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = &newFrontendIPConfig
	return nil
}
func RemoveLoadbalancerProbeConfiguration(lb *mgmtnetwork.LoadBalancer, resourceID string) error {
	newProbeConfiguration := make([]mgmtnetwork.Probe, 0)
	for _, probe := range *lb.LoadBalancerPropertiesFormat.Probes {
		if strings.EqualFold(*probe.ID, resourceID) {
			if isProbeInUse(probe) {
				return fmt.Errorf("probe %s still in use by load balancing rules, remove references prior to removing the probe", resourceID)
			}
			continue
		}
		newProbeConfiguration = append(newProbeConfiguration, probe)
	}
	lb.LoadBalancerPropertiesFormat.Probes = &newProbeConfiguration
	return nil
}

func isProbeInUse(probe mgmtnetwork.Probe) bool {
	return probe.LoadBalancingRules != nil
}

func isFrontendIPConfigReferenced(fipConfig mgmtnetwork.FrontendIPConfiguration) bool {
	return fipConfig.LoadBalancingRules != nil || fipConfig.InboundNatPools != nil || fipConfig.InboundNatRules != nil || fipConfig.OutboundRules != nil
}
