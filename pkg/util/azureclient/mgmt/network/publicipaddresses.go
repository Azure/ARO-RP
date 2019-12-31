package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
)

// PublicIPAddressesClient is a minimal interface for azure PublicIPAddressesClient
type PublicIPAddressesClient interface {
	Get(ctx context.Context, resourceGroupName string, publicIPAddressName string, expand string) (result network.PublicIPAddress, err error)
	List(ctx context.Context, resourceGroupName string) (ips []network.PublicIPAddress, err error)
	PublicIPAddressesClientAddons
}

type publicIPAddressesClient struct {
	network.PublicIPAddressesClient
}

var _ PublicIPAddressesClient = &publicIPAddressesClient{}

// NewPublicIPAddressesClient creates a new PublicIPAddressesClient
func NewPublicIPAddressesClient(subscriptionID string, authorizer autorest.Authorizer) PublicIPAddressesClient {
	client := network.NewPublicIPAddressesClient(subscriptionID)
	client.Authorizer = authorizer

	return &publicIPAddressesClient{
		PublicIPAddressesClient: client,
	}
}
