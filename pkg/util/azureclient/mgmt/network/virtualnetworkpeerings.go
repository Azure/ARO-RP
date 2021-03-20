package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type VirtualNetworkPeeringsClient interface {
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

	return &virtualNetworkPeeringsClient{
		VirtualNetworkPeeringsClient: client,
	}
}
