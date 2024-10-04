package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type VirtualNetworkPeeringsClient interface {
	VirtualNetworkPeeringsAddons
}

type virtualNetworkPeeringsClient struct {
	*armnetwork.VirtualNetworkPeeringsClient
}

// NewVirtualNetworkPeeringsClient creates a new VirtualNetworkPeeringsClient
func NewVirtualNetworkPeeringsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (VirtualNetworkPeeringsClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &virtualNetworkPeeringsClient{clientFactory.NewVirtualNetworkPeeringsClient()}, err
}
