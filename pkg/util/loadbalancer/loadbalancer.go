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
	if fipConfig == nil || fipConfig.Properties == nil {
		return false
	}

	return len(fipConfig.Properties.LoadBalancingRules) > 0 ||
		len(fipConfig.Properties.InboundNatPools) > 0 ||
		len(fipConfig.Properties.InboundNatRules) > 0 ||
		len(fipConfig.Properties.OutboundRules) > 0
}

func RemoveLoadBalancingRule(lb *armnetwork.LoadBalancer, resourceID string) error {
	if lb.Properties == nil {
		return nil
	}

	lb.Properties.LoadBalancingRules = removeLoadBalancingRulesByID(lb.Properties.LoadBalancingRules, resourceID)

	for _, fipConfig := range lb.Properties.FrontendIPConfigurations {
		if fipConfig == nil || fipConfig.Properties == nil {
			continue
		}
		fipConfig.Properties.LoadBalancingRules = removeSubResourcesByID(fipConfig.Properties.LoadBalancingRules, resourceID)
	}

	for _, backendPool := range lb.Properties.BackendAddressPools {
		if backendPool == nil || backendPool.Properties == nil {
			continue
		}
		backendPool.Properties.LoadBalancingRules = removeSubResourcesByID(backendPool.Properties.LoadBalancingRules, resourceID)
	}

	for _, probe := range lb.Properties.Probes {
		if probe == nil || probe.Properties == nil {
			continue
		}
		probe.Properties.LoadBalancingRules = removeSubResourcesByID(probe.Properties.LoadBalancingRules, resourceID)
	}

	return nil
}

func RemoveHealthProbe(lb *armnetwork.LoadBalancer, resourceID string) error {
	newProbes := make([]*armnetwork.Probe, 0)

	// Iterate over probes, build a list without the targeted resourceID, if possible
	for _, probe := range lb.Properties.Probes {
		if probe.ID != nil && strings.EqualFold(*probe.ID, resourceID) {
			if probe.Properties != nil && len(probe.Properties.LoadBalancingRules) > 0 {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("Load balancer health probe %s is used by load balancing rules, remove the referencing load balancing rules before removing the health probe", resourceID))
			}
			continue
		}
		newProbes = append(newProbes, probe)
	}
	lb.Properties.Probes = newProbes

	// return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeInvalidParameter, "", "No health probes found for this loadbalancer")

	return nil
}

func removeLoadBalancingRulesByID(rules []*armnetwork.LoadBalancingRule, resourceID string) []*armnetwork.LoadBalancingRule {
	if rules == nil {
		return nil
	}

	filteredRules := make([]*armnetwork.LoadBalancingRule, 0, len(rules))
	removed := false
	for _, rule := range rules {
		if rule != nil && matchesLoadBalancingRule(rule, resourceID) {
			removed = true
			continue
		}
		filteredRules = append(filteredRules, rule)
	}

	if !removed {
		return rules
	}

	return filteredRules
}

func removeSubResourcesByID(subResources []*armnetwork.SubResource, resourceID string) []*armnetwork.SubResource {
	if subResources == nil {
		return nil
	}

	filteredSubResources := make([]*armnetwork.SubResource, 0, len(subResources))
	removed := false
	for _, subResource := range subResources {
		if subResource != nil && matchesSubResource(subResource, resourceID) {
			removed = true
			continue
		}
		filteredSubResources = append(filteredSubResources, subResource)
	}

	if !removed {
		return subResources
	}

	return filteredSubResources
}

func matchesLoadBalancingRule(rule *armnetwork.LoadBalancingRule, resourceID string) bool {
	targetName := lastToken(resourceID)

	return (rule.ID != nil && strings.EqualFold(*rule.ID, resourceID)) ||
		(rule.ID != nil && strings.EqualFold(lastToken(*rule.ID), targetName)) ||
		(rule.Name != nil && strings.EqualFold(*rule.Name, targetName))
}

func matchesSubResource(subResource *armnetwork.SubResource, resourceID string) bool {
	if subResource.ID == nil {
		return false
	}

	return strings.EqualFold(*subResource.ID, resourceID) ||
		strings.EqualFold(lastToken(*subResource.ID), lastToken(resourceID))
}

func lastToken(resourceID string) string {
	if i := strings.LastIndexByte(resourceID, '/'); i != -1 {
		return resourceID[i+1:]
	}

	return resourceID
}
