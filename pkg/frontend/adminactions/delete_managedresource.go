package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	armsdk "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/loadbalancer"
)

var denyList = []string{
	`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Network/privateLinkServices/([^/]+)$`,
	`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Network/privateEndpoints/([^/]+)$`,
	`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Storage/(.+)$`,
}

func (a *azureActions) ResourceDeleteAndWait(ctx context.Context, resourceID string) error {
	idParts, err := armsdk.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	for _, regex := range denyList {
		re := regexp.MustCompile(regex)
		if re.MatchString(resourceID) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("deletion of resource %s is forbidden", resourceID))
		}
	}

	apiVersion := azureclient.APIVersion(strings.ToLower(idParts.ResourceType.String()))
	resourceType := strings.ToLower(idParts.ResourceType.String())

	_, err = a.resources.GetByID(ctx, resourceID, apiVersion)
	if err != nil {
		return err
	}

	// FrontendIPConfiguration cannot be deleted with DeleteByIDAndWait (DELETE method is invalid on frontendIPConfiguration resourceID)
	if resourceType == "microsoft.network/loadbalancers/frontendipconfigurations" {
		return a.deleteFrontendIPConfiguration(ctx, resourceID, idParts.ResourceGroupName, idParts.Parent.Name)
	}

	// HealthProbes cannot be deleted with DeleteByIDAndWait either.
	if resourceType == "microsoft.network/loadbalancers/probes" {
		return a.deleteHealthProbe(ctx, resourceID, idParts.ResourceGroupName, idParts.Parent.Name)
	}

	// LoadBalancingRules must be removed via an update to the parent load balancer.
	if resourceType == "microsoft.network/loadbalancers/loadbalancingrules" {
		return a.deleteLoadBalancingRule(ctx, resourceID, idParts.ResourceGroupName, idParts.Parent.Name)
	}

	return arm.RetryableDelete(ctx, func() error {
		return a.resources.DeleteByIDAndWait(ctx, resourceID, apiVersion)
	}, a.log, "deleting resource "+resourceID)
}

func (a *azureActions) deleteFrontendIPConfiguration(ctx context.Context, resourceID string, rg string, loadBalancerName string) error {
	lb, err := a.loadBalancers.Get(ctx, rg, loadBalancerName, nil)
	if err != nil {
		return err
	}

	err = loadbalancer.RemoveFrontendIPConfiguration(&lb.LoadBalancer, resourceID)
	if err != nil {
		return err
	}

	return arm.Retryable(ctx, func() error {
		return a.loadBalancers.CreateOrUpdateAndWait(ctx, rg, loadBalancerName, lb.LoadBalancer, nil)
	}, a.log, "deleting frontend IP configuration from load balancer "+loadBalancerName)
}

func (a *azureActions) deleteHealthProbe(ctx context.Context, resourceID string, rg string, loadBalancerName string) error {
	lb, err := a.loadBalancers.Get(ctx, rg, loadBalancerName, nil)
	if err != nil {
		return err
	}

	err = loadbalancer.RemoveHealthProbe(&lb.LoadBalancer, resourceID)
	if err != nil {
		return err
	}

	return arm.Retryable(ctx, func() error {
		return a.loadBalancers.CreateOrUpdateAndWait(ctx, rg, loadBalancerName, lb.LoadBalancer, nil)
	}, a.log, "deleting health probe from load balancer "+loadBalancerName)
}

func (a *azureActions) deleteLoadBalancingRule(ctx context.Context, resourceID string, rg string, loadBalancerName string) error {
	lb, err := a.loadBalancers.Get(ctx, rg, loadBalancerName, nil)
	if err != nil {
		return err
	}

	err = loadbalancer.RemoveLoadBalancingRule(&lb.LoadBalancer, resourceID)
	if err != nil {
		return err
	}

	return arm.Retryable(ctx, func() error {
		return a.loadBalancers.CreateOrUpdateAndWait(ctx, rg, loadBalancerName, lb.LoadBalancer, nil)
	}, a.log, "deleting load balancing rule from load balancer "+loadBalancerName)
}
