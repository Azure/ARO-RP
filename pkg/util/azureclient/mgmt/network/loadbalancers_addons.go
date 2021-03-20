package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

// LoadBalancersClientAddons contains addons for Azure LoadBalancersClient
type LoadBalancersClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, loadBalancerName string, parameters mgmtnetwork.LoadBalancer) error
}

func (c *loadBalancersClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, loadBalancerName string, parameters mgmtnetwork.LoadBalancer) error {
	future, err := c.LoadBalancersClient.CreateOrUpdate(ctx, resourceGroupName, loadBalancerName, parameters)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, c.Client)
}
