package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
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
func NewSubnetsClient(subscriptionID string, authorizer autorest.Authorizer) SubnetsClient {
	client := mgmtnetwork.NewSubnetsClient(subscriptionID)
	client.Authorizer = authorizer

	return &subnetsClient{
		SubnetsClient: client,
	}
}
