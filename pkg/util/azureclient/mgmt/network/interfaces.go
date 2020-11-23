package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// InterfacesClient is a minimal interface for azure InterfacesClient
type InterfacesClient interface {
	InterfacesClientAddons
}

type interfacesClient struct {
	mgmtnetwork.InterfacesClient
}

var _ InterfacesClient = &interfacesClient{}

// NewInterfacesClient creates a new InterfacesClient
func NewInterfacesClient(environment *azure.Environment, subscriptionID string, authorizer autorest.Authorizer) InterfacesClient {
	client := mgmtnetwork.NewInterfacesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &interfacesClient{
		InterfacesClient: client,
	}
}
