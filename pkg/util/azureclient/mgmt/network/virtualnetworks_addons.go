package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
)

// VirtualNetworksClientAddons contains addons for VirtualNetworksClient
type VirtualNetworksClientAddons interface {
	List(ctx context.Context, resourceGroupName string) (virtualnetworks []network.VirtualNetwork, err error)
}

func (c *virtualNetworksClient) List(ctx context.Context, resourceGroupName string) (virtualnetworks []network.VirtualNetwork, err error) {
	page, err := c.VirtualNetworksClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		virtualnetworks = append(virtualnetworks, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return virtualnetworks, nil
}
