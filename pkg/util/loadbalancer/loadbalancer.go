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

func isFrontendIPConfigReferenced(fipConfig mgmtnetwork.FrontendIPConfiguration) bool {
	return fipConfig.LoadBalancingRules != nil || fipConfig.InboundNatPools != nil || fipConfig.InboundNatRules != nil || fipConfig.OutboundRules != nil
}
