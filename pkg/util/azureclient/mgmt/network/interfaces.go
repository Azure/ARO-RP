package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
)

// InterfacesClient is a minimal interface for azure InterfacesClient
type InterfacesClient interface {
	InterfacesClientAddons
	Get(ctx context.Context, resourceGroupName string, networkInterfaceName string, expand string) (result mgmtnetwork.Interface, err error)
}

type interfacesClient struct {
	mgmtnetwork.InterfacesClient
}

var _ InterfacesClient = &interfacesClient{}

// NewInterfacesClient creates a new InterfacesClient
func NewInterfacesClient(subscriptionID string, authorizer autorest.Authorizer) InterfacesClient {
	client := mgmtnetwork.NewInterfacesClient(subscriptionID)
	client.Authorizer = authorizer

	return &interfacesClient{
		InterfacesClient: client,
	}
}
