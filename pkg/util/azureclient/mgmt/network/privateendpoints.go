package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// PrivateEndpointsClient is a minimal interface for azure PrivateEndpointsClient
type PrivateEndpointsClient interface {
	Get(ctx context.Context, resourceGroupName string, privateEndpointName string, expand string) (result mgmtnetwork.PrivateEndpoint, err error)
	PrivateEndpointsClientAddons
}

type privateEndpointsClient struct {
	mgmtnetwork.PrivateEndpointsClient
}

var _ PrivateEndpointsClient = &privateEndpointsClient{}

// NewPrivateEndpointsClient creates a new PrivateEndpointsClient
func NewPrivateEndpointsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) PrivateEndpointsClient {
	client := mgmtnetwork.NewPrivateEndpointsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &privateEndpointsClient{
		PrivateEndpointsClient: client,
	}
}
