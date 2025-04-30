package loadbalancer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

func RemoveFrontendIPConfiguration(lb *armnetwork.LoadBalancer, resourceID string) error {
	newFrontendIPConfig := make([]*armnetwork.FrontendIPConfiguration, 0, len(lb.Properties.FrontendIPConfigurations))
	for _, fipConfig := range lb.Properties.FrontendIPConfigurations {
		if strings.EqualFold(*fipConfig.ID, resourceID) {
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
