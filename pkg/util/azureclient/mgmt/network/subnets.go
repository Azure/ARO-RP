package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// SubnetsClient is a minimal interface for azure SubnetsClient
type SubnetsClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, expand string) (result mgmtnetwork.Subnet, err error)
	SubnetsClientAddons
}

type subnetsClient struct {
	mgmtnetwork.SubnetsClient
}

var _ SubnetsClient = &subnetsClient{}

// NewSubnetsClient creates a new SubnetsClient
func NewSubnetsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) SubnetsClient {
	client := mgmtnetwork.NewSubnetsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &subnetsClient{
		SubnetsClient: client,
	}
}
