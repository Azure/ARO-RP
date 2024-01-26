package loadbalancer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const OutboundRuleV4 = "outbound-rule-v4"

func RemoveFrontendIPConfiguration(lb *mgmtnetwork.LoadBalancer, resourceID string) error {
	if lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations == nil {
		return fmt.Errorf("FrontendIPConfigurations are nil")
	}

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

// Remove outbound-rule-v4 IPs and corresponding frontendIPConfig from load balancer
func RemoveOutboundIPsFromLB(lb mgmtnetwork.LoadBalancer) {
	removeOutboundRuleV4FrontendIPConfig(lb)
	setOutboundRuleV4(lb, []mgmtnetwork.SubResource{})
}

func removeOutboundRuleV4FrontendIPConfig(lb mgmtnetwork.LoadBalancer) {
	var savedFIPConfig = make([]mgmtnetwork.FrontendIPConfiguration, 0, len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations))
	var outboundRuleFrontendConfig = getOutboundRuleV4FIPConfigs(lb)

	for i := 0; i < len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations); i++ {
		fipConfigID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].ID
		fipConfig := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i]
		hasLBRules := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].LoadBalancingRules != nil
		if _, ok := outboundRuleFrontendConfig[fipConfigID]; ok && !hasLBRules {
			continue
		}
		savedFIPConfig = append(savedFIPConfig, fipConfig)
	}
	lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = &savedFIPConfig
}

func getOutboundRuleV4FIPConfigs(lb mgmtnetwork.LoadBalancer) map[string]mgmtnetwork.SubResource {
	var obRuleV4FIPConfigs = make(map[string]mgmtnetwork.SubResource)
	for _, obRule := range *lb.LoadBalancerPropertiesFormat.OutboundRules {
		if *obRule.Name == OutboundRuleV4 {
			for i := 0; i < len(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations); i++ {
				fipConfigID := *(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations)[i].ID
				fipConfig := (*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations)[i]
				obRuleV4FIPConfigs[fipConfigID] = fipConfig
			}
			break
		}
	}
	return obRuleV4FIPConfigs
}

// Returns a map of Frontend IP Configurations.  Frontend IP Configurations can be looked up by Public IP Address ID or Frontend IP Configuration ID
func getFrontendIPConfigs(lb mgmtnetwork.LoadBalancer) map[string]mgmtnetwork.FrontendIPConfiguration {
	var frontendIPConfigs = make(map[string]mgmtnetwork.FrontendIPConfiguration, len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations))

	for i := 0; i < len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations); i++ {
		fipConfigID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].ID
		fipConfigIPAddressID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].FrontendIPConfigurationPropertiesFormat.PublicIPAddress.ID
		fipConfig := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i]
		frontendIPConfigs[fipConfigID] = fipConfig
		frontendIPConfigs[fipConfigIPAddressID] = fipConfig
	}

	return frontendIPConfigs
}

// Adds IPs or IPPrefixes to the load balancer outbound rule "outbound-rule-v4".
func AddOutboundIPsToLB(resourceGroupID string, lb mgmtnetwork.LoadBalancer, obIPsOrIPPrefixes []api.ResourceReference) {
	frontendIPConfigs := getFrontendIPConfigs(lb)
	outboundRuleV4FrontendIPConfig := []mgmtnetwork.SubResource{}

	// add IP Addresses to frontendConfig
	for _, obIPOrIPPrefix := range obIPsOrIPPrefixes {
		// check if the frontend config exists in the map to avoid duplicate entries
		if _, ok := frontendIPConfigs[obIPOrIPPrefix.ID]; !ok {
			frontendIPConfigName := stringutils.LastTokenByte(obIPOrIPPrefix.ID, '/')
			frontendConfigID := fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", resourceGroupID, *lb.Name, frontendIPConfigName)
			*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = append(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations, NewFrontendIPConfig(frontendIPConfigName, frontendConfigID, obIPOrIPPrefix.ID))
			outboundRuleV4FrontendIPConfig = append(outboundRuleV4FrontendIPConfig, NewOutboundRuleFrontendIPConfig(frontendConfigID))
		} else {
			// frontendIPConfig already exists and just needs to be added to the outbound rule
			frontendConfig := frontendIPConfigs[obIPOrIPPrefix.ID]
			outboundRuleV4FrontendIPConfig = append(outboundRuleV4FrontendIPConfig, NewOutboundRuleFrontendIPConfig(*frontendConfig.ID))
		}
	}

	setOutboundRuleV4(lb, outboundRuleV4FrontendIPConfig)
}

func GetOutboundIPsFromLB(lb mgmtnetwork.LoadBalancer) []api.ResourceReference {
	var outboundIPs []api.ResourceReference
	fipConfigs := getFrontendIPConfigs(lb)

	for _, obRule := range *lb.LoadBalancerPropertiesFormat.OutboundRules {
		if *obRule.Name == OutboundRuleV4 {
			for i := 0; i < len(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations); i++ {
				id := *(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations)[i].ID
				if fipConfig, ok := fipConfigs[id]; ok {
					outboundIPs = append(outboundIPs, api.ResourceReference{ID: *fipConfig.PublicIPAddress.ID})
				}
			}
		}
	}

	return outboundIPs
}

func setOutboundRuleV4(lb mgmtnetwork.LoadBalancer, outboundRuleV4FrontendIPConfig []mgmtnetwork.SubResource) {
	for _, outboundRule := range *lb.LoadBalancerPropertiesFormat.OutboundRules {
		if *outboundRule.Name == OutboundRuleV4 {
			outboundRule.OutboundRulePropertiesFormat.FrontendIPConfigurations = &outboundRuleV4FrontendIPConfig
			break
		}
	}
}

func NewFrontendIPConfig(name string, id string, publicIPorIPPrefixID string) mgmtnetwork.FrontendIPConfiguration {
	// TODO: add check for publicIPorIPPrefixID
	return mgmtnetwork.FrontendIPConfiguration{
		Name: &name,
		ID:   &id,
		FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
			PublicIPAddress: &mgmtnetwork.PublicIPAddress{
				ID: &publicIPorIPPrefixID,
			},
		},
	}
}

func NewOutboundRuleFrontendIPConfig(id string) mgmtnetwork.SubResource {
	return mgmtnetwork.SubResource{
		ID: &id,
	}
}
