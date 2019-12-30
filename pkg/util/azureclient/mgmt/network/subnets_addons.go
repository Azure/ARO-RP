package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
)

// SubnetsClientAddons contains addons for SubnetsClient
type SubnetsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, subnetParameters network.Subnet) (err error)
}

func (c *subnetsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, subnetParameters network.Subnet) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, virtualNetworkName, subnetName, subnetParameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
