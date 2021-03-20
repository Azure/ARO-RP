package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

// SubnetsClientAddons contains addons for SubnetsClient
type SubnetsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, subnetParameters mgmtnetwork.Subnet) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string) error
}

func (c *subnetsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, subnetParameters mgmtnetwork.Subnet) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, virtualNetworkName, subnetName, subnetParameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *subnetsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string) error {
	future, err := c.Delete(ctx, resourceGroupName, virtualNetworkName, subnetName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
