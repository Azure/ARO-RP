package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

type VirtualNetworkPeeringsAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string, virtualNetworkPeeringParameters mgmtnetwork.VirtualNetworkPeering) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string) (err error)
}

func (c *virtualNetworkPeeringsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string, virtualNetworkPeeringParameters mgmtnetwork.VirtualNetworkPeering) (err error) {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, virtualNetworkName, virtualNetworkPeeringName, virtualNetworkPeeringParameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualNetworkPeeringsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string) (err error) {
	future, err := c.VirtualNetworkPeeringsClient.Delete(ctx, resourceGroupName, virtualNetworkName, virtualNetworkPeeringName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
