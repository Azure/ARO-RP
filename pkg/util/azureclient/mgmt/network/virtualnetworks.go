package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// VirtualNetworksClient is a minimal interface for azure VirtualNetworksClient
type VirtualNetworksClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, expand string) (vnet mgmtnetwork.VirtualNetwork, err error)
}

// Makes sure the interface is implemented, not very useful since it will error out other places.
var _ VirtualNetworksClient = mgmtnetwork.VirtualNetworksClient{}

// NewVirtualNetworksClient creates a new VirtualNetworksClient
func NewVirtualNetworksClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) mgmtnetwork.VirtualNetworksClient {
	client := mgmtnetwork.NewVirtualNetworksClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return client
}
