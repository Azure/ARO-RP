package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// SubnetsClientAddons contains addons for SubnetsClient
type SubnetsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, virtualNetworkName, subnetName string, subnetParameters armnetwork.Subnet, options *armnetwork.SubnetsClientBeginCreateOrUpdateOptions) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName, virtualNetworkName, subnetName string, options *armnetwork.SubnetsClientBeginDeleteOptions) error
}

func (c *subnetsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, virtualNetworkName, subnetName string, subnetParameters armnetwork.Subnet, options *armnetwork.SubnetsClientBeginCreateOrUpdateOptions) error {
	poller, err := c.SubnetsClient.BeginCreateOrUpdate(ctx, resourceGroupName, virtualNetworkName, subnetName, subnetParameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *subnetsClient) DeleteAndWait(ctx context.Context, resourceGroupName, virtualNetworkName, subnetName string, options *armnetwork.SubnetsClientBeginDeleteOptions) error {
	poller, err := c.SubnetsClient.BeginDelete(ctx, resourceGroupName, virtualNetworkName, subnetName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
