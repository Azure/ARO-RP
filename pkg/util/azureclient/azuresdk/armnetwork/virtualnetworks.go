package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// VirtualNetworksClient is a minimal interface for azure VirtualNetworksClient
type VirtualNetworksClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientGetOptions) (vnet armnetwork.VirtualNetworksClientGetResponse, err error)
}

type virtualNetworksClient struct {
	*armnetwork.VirtualNetworksClient
}

// NewVirtualNetworksClient creates a new VirtualNetworksClient
func NewVirtualNetworksClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (VirtualNetworksClient, error) {
	client, err := armnetwork.NewVirtualNetworksClient(subscriptionID, credential, options)

	return &virtualNetworksClient{client}, err
}
