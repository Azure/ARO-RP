package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
)

// VirtualNetworksClient is a minimal interface for azure VirtualNetworksClient
type VirtualNetworksClient interface {
	VirtualNetworksClientAddons
}

type virtualNetworksClient struct {
	network.VirtualNetworksClient
}

var _ VirtualNetworksClient = &virtualNetworksClient{}

// NewVirtualNetworksClient creates a new VirtualNetworksClient
func NewVirtualNetworksClient(subscriptionID string, authorizer autorest.Authorizer) VirtualNetworksClient {
	client := network.NewVirtualNetworksClient(subscriptionID)
	client.Authorizer = authorizer

	return &virtualNetworksClient{
		VirtualNetworksClient: client,
	}
}
