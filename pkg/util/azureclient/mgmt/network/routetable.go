package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// RouteTablesClient is a minimal interface for azure RouteTablesClient
type RouteTablesClient interface {
	Get(ctx context.Context, resourceGroupName string, routeTableName string, expand string) (result mgmtnetwork.RouteTable, err error)
	RouteTablesClientAddons
}

type routeTablesClient struct {
	mgmtnetwork.RouteTablesClient
}

var _ RouteTablesClient = &routeTablesClient{}

// NewRouteTablesClient creates a new RouteTablesClient
func NewRouteTablesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) RouteTablesClient {
	client := mgmtnetwork.NewRouteTablesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &routeTablesClient{
		RouteTablesClient: client,
	}
}
