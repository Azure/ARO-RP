package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/loadbalancer"
)

var (
	denyList = []string{
		`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Network/privateLinkServices/([^/]+)$`,
		`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Network/privateEndpoints/([^/]+)$`,
		`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Storage/(.+)$`,
	}
)

func (a *azureActions) ResourceDeleteAndWait(ctx context.Context, resourceID string) error {
	idParts, err := arm.ParseResourceID(resourceID)
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

	_, err = a.resources.GetByID(ctx, resourceID, apiVersion)
	if err != nil {
		return err
	}

	// FrontendIPConfiguration cannot be deleted with DeleteByIDAndWait (DELETE method is invalid on frontendIPConfiguration resourceID)
	if idParts.ResourceType.String() == "Microsoft.Network/loadBalancers/frontendIPConfigurations" {
		return a.deleteFrontendIPConfiguration(ctx, resourceID, idParts.ResourceGroupName, idParts.Parent.Name)
	}

	// HealthProbes cannot be deleted with DeleteByIDAndWait either.
	if idParts.ResourceType.String() == "Microsoft.Network/loadBalancers/probes" {
		return a.deleteHealthProbe(ctx, resourceID, idParts.ResourceGroupName, idParts.Parent.Name)
	}

	return a.resources.DeleteByIDAndWait(ctx, resourceID, apiVersion)
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

	return a.loadBalancers.CreateOrUpdateAndWait(ctx, rg, loadBalancerName, lb.LoadBalancer, nil)
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

	return a.loadBalancers.CreateOrUpdateAndWait(ctx, rg, loadBalancerName, lb.LoadBalancer, nil)
}
