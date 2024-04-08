package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type VirtualNetworkPeeringsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string, virtualNetworkPeeringParameters mgmtnetwork.VirtualNetworkPeering) (result mgmtnetwork.VirtualNetworkPeeringsCreateOrUpdateFuture, err error)
	Delete(ctx context.Context, resourceGroupName string, virtualNetworkName string, virtualNetworkPeeringName string) (result mgmtnetwork.VirtualNetworkPeeringsDeleteFuture, err error)
	VirtualNetworkPeeringsAddons
}

type virtualNetworkPeeringsClient struct {
	mgmtnetwork.VirtualNetworkPeeringsClient
}

var _ VirtualNetworkPeeringsClient = &virtualNetworkPeeringsClient{}

// NewVirtualNetworkPeeringsClient creates a new VirtualNetworkPeeringsClient
func NewVirtualNetworkPeeringsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) VirtualNetworkPeeringsClient {
	client := mgmtnetwork.NewVirtualNetworkPeeringsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &virtualNetworkPeeringsClient{
		VirtualNetworkPeeringsClient: client,
	}
}
