package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// PrivateLinkServicesClient is a minimal interface for azure PrivateLinkServicesClient
type PrivateLinkServicesClient interface {
	DeletePrivateEndpointConnection(ctx context.Context, resourceGroupName string, serviceName string, peConnectionName string) (result mgmtnetwork.PrivateLinkServicesDeletePrivateEndpointConnectionFuture, err error)
	Get(ctx context.Context, resourceGroupName string, serviceName string, expand string) (result mgmtnetwork.PrivateLinkService, err error)
	UpdatePrivateEndpointConnection(ctx context.Context, resourceGroupName string, serviceName string, peConnectionName string, parameters mgmtnetwork.PrivateEndpointConnection) (result mgmtnetwork.PrivateEndpointConnection, err error)
	PrivateLinkServicesClientAddons
}

type privateLinkServicesClient struct {
	mgmtnetwork.PrivateLinkServicesClient
}

var _ PrivateLinkServicesClient = &privateLinkServicesClient{}

// NewPrivateLinkServicesClient creates a new PrivateLinkServicesClient
func NewPrivateLinkServicesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) PrivateLinkServicesClient {
	client := mgmtnetwork.NewPrivateLinkServicesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &privateLinkServicesClient{
		PrivateLinkServicesClient: client,
	}
}
