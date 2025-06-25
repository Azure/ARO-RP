package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

// PublicIPAddressesClient is a minimal interface for azure PublicIPAddressesClient
type PublicIPAddressesClient interface {
	Get(ctx context.Context, resourceGroupName string, publicIPAddressName string, options *armnetwork.PublicIPAddressesClientGetOptions) (result armnetwork.PublicIPAddressesClientGetResponse, err error)
	PublicIPAddressesClientAddons
}

type publicIPAddressesClient struct {
	*armnetwork.PublicIPAddressesClient
}

// NewPublicIPAddressesClient creates a new PublicIPAddressesClient
func NewPublicIPAddressesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (PublicIPAddressesClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &publicIPAddressesClient{clientFactory.NewPublicIPAddressesClient()}, nil
}
