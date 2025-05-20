package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

// LoadBalancersClientAddons contains addons for Azure LoadBalancersClient
type LoadBalancersClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, loadBalancerName string, parameters armnetwork.LoadBalancer, options *armnetwork.LoadBalancersClientBeginCreateOrUpdateOptions) error
}

func (c *loadBalancersClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, loadBalancerName string, parameters armnetwork.LoadBalancer, options *armnetwork.LoadBalancersClientBeginCreateOrUpdateOptions) error {
	poller, err := c.LoadBalancersClient.BeginCreateOrUpdate(ctx, resourceGroupName, loadBalancerName, parameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
