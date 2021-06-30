package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// VirtualNetworksClient is a minimal interface for azure VirtualNetworksClient
type VirtualNetworksClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters mgmtnetwork.VirtualNetwork) (result mgmtnetwork.VirtualNetworksCreateOrUpdateFuture, err error)
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, expand string) (vnet mgmtnetwork.VirtualNetwork, err error)
	VirtualNetworksClientAddons
}

type virtualNetworksClient struct {
	mgmtnetwork.VirtualNetworksClient
}

var _ VirtualNetworksClient = &virtualNetworksClient{}

// NewVirtualNetworksClient creates a new VirtualNetworksClient
func NewVirtualNetworksClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) VirtualNetworksClient {
	client := mgmtnetwork.NewVirtualNetworksClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &virtualNetworksClient{
		VirtualNetworksClient: client,
	}
}
