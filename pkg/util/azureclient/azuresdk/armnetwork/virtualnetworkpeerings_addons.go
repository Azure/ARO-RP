package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

type VirtualNetworkPeeringsAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string, virtualNetworkPeeringParameters armnetwork.VirtualNetworkPeering, options *armnetwork.VirtualNetworkPeeringsClientBeginCreateOrUpdateOptions) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string, options *armnetwork.VirtualNetworkPeeringsClientBeginDeleteOptions) (err error)
}

func (c *virtualNetworkPeeringsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string, virtualNetworkPeeringParameters armnetwork.VirtualNetworkPeering, options *armnetwork.VirtualNetworkPeeringsClientBeginCreateOrUpdateOptions) (err error) {
	poller, err := c.VirtualNetworkPeeringsClient.BeginCreateOrUpdate(ctx, resourceGroupName, virtualNetworkName, virtualNetworkPeeringName, virtualNetworkPeeringParameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *virtualNetworkPeeringsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string, options *armnetwork.VirtualNetworkPeeringsClientBeginDeleteOptions) (err error) {
	poller, err := c.VirtualNetworkPeeringsClient.BeginDelete(ctx, resourceGroupName, virtualNetworkName, virtualNetworkPeeringName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
