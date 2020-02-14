package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
)

// VirtualNetworksClientAddons contains addons for VirtualNetworksClient
type VirtualNetworksClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters network.VirtualNetwork) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string) (err error)
	List(ctx context.Context, resourceGroupName string) (virtualnetworks []network.VirtualNetwork, err error)
}

func (c *virtualNetworksClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters network.VirtualNetwork) (err error) {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, virtualNetworkName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualNetworksClient) DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string) (err error) {
	future, err := c.Delete(ctx, resourceGroupName, virtualNetworkName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualNetworksClient) List(ctx context.Context, resourceGroupName string) (virtualnetworks []network.VirtualNetwork, err error) {
	page, err := c.VirtualNetworksClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		virtualnetworks = append(virtualnetworks, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return virtualnetworks, nil
}
