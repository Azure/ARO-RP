package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/loadbalancer"
)

var (
	frontendIPConfigurationPattern = `(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Network/loadBalancers/(.+)/frontendIPConfigurations/([^/]+)$`
	denyList                       = []string{
		`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Network/privateLinkServices/([^/]+)$`,
		`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Network/privateEndpoints/([^/]+)$`,
		`(?i)^/subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft\.Storage/(.+)$`,
	}
)

func (a *azureActions) ResourceDeleteAndWait(ctx context.Context, resourceID string) error {
	idParts, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	for _, regex := range denyList {
		re := regexp.MustCompile(regex)
		if re.MatchString(resourceID) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("deletion of resource %s is forbidden", resourceID))
		}
	}

	apiVersion := azureclient.APIVersion(strings.ToLower(idParts.Provider + "/" + idParts.ResourceType))

	_, err = a.resources.GetByID(ctx, resourceID, apiVersion)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(frontendIPConfigurationPattern)
	// FrontendIPConfiguration cannot be deleted with DeleteByIDAndWait (DELETE method is invalid on frontendIPConfiguration resourceID)
	if re.MatchString(resourceID) {
		return a.deleteFrontendIPConfiguration(ctx, resourceID)
	}

	return a.resources.DeleteByIDAndWait(ctx, resourceID, apiVersion)
}

func (a *azureActions) deleteFrontendIPConfiguration(ctx context.Context, resourceID string) error {
	idParts := strings.Split(resourceID, "/")
	rg := idParts[4]
	lbName := idParts[8]

	lb, err := a.loadBalancers.Get(ctx, rg, lbName, nil)
	if err != nil {
		return err
	}

	err = loadbalancer.RemoveFrontendIPConfiguration(&lb.LoadBalancer, resourceID)
	if err != nil {
		return err
	}

	return a.loadBalancers.CreateOrUpdateAndWait(ctx, rg, lbName, lb.LoadBalancer, nil)
}
