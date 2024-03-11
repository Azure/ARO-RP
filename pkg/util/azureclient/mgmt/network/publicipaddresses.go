package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// PublicIPAddressesClient is a minimal interface for azure PublicIPAddressesClient
type PublicIPAddressesClient interface {
	Get(ctx context.Context, resourceGroupName string, publicIPAddressName string, expand string) (result mgmtnetwork.PublicIPAddress, err error)
	PublicIPAddressesClientAddons
}

type publicIPAddressesClient struct {
	mgmtnetwork.PublicIPAddressesClient
}

var _ PublicIPAddressesClient = &publicIPAddressesClient{}

// NewPublicIPAddressesClient creates a new PublicIPAddressesClient
func NewPublicIPAddressesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) PublicIPAddressesClient {
	client := mgmtnetwork.NewPublicIPAddressesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &publicIPAddressesClient{
		PublicIPAddressesClient: client,
	}
}
