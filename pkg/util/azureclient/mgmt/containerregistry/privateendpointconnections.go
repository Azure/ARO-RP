package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2019-12-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
)

// PrivateEndpointConnectionsClient is a minimal interface for azure PrivateEndpointConnectionsClient
type PrivateEndpointConnectionsClient interface {
	PrivateEndpointConnectionsClientAddons
	CreateOrUpdate(ctx context.Context, resourceGroupName string, registryName string, privateEndpointConnectionName string, privateEndpointConnection mgmtcontainerregistry.PrivateEndpointConnection) (result mgmtcontainerregistry.PrivateEndpointConnectionsCreateOrUpdateFuture, err error)
}

type privateEndpointConnectionsClient struct {
	mgmtcontainerregistry.PrivateEndpointConnectionsClient
}

var _ PrivateEndpointConnectionsClient = &privateEndpointConnectionsClient{}

// NewPrivateEndpointConnectionsClient creates a new NewPrivateEndpointConnectionsClient
func NewPrivateEndpointConnectionsClient(subscriptionID string, authorizer autorest.Authorizer) PrivateEndpointConnectionsClient {
	client := mgmtcontainerregistry.NewPrivateEndpointConnectionsClient(subscriptionID)
	client.Authorizer = authorizer

	return &privateEndpointConnectionsClient{
		PrivateEndpointConnectionsClient: client,
	}
}
