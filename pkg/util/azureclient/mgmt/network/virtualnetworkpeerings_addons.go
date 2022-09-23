package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

type VirtualNetworkPeeringsAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string) (err error)
}

func (c *virtualNetworkPeeringsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string) (err error) {
	future, err := c.VirtualNetworkPeeringsClient.Delete(ctx, resourceGroupName, virtualNetworkName, virtualNetworkPeeringName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
