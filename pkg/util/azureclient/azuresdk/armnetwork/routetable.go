package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// RouteTablesClient is a minimal interface for azure RouteTablesClient
type RouteTablesClient interface {
	Get(ctx context.Context, resourceGroupName string, routeTableName string, options *armnetwork.RouteTablesClientGetOptions) (result armnetwork.RouteTablesClientGetResponse, err error)
	RouteTablesClientAddons
}

type routeTablesClient struct {
	*armnetwork.RouteTablesClient
}

// NewRouteTablesClient creates a new RouteTablesClient
func NewRouteTablesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (RouteTablesClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &routeTablesClient{clientFactory.NewRouteTablesClient()}, err
}
