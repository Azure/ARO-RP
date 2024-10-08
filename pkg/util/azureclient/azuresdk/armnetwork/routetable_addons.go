package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// RouteTablesClientAddons contains addons for RouteTablesClient
type RouteTablesClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string, routeTableName string, options *armnetwork.RouteTablesClientBeginDeleteOptions) error
}

func (c *routeTablesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, routeTableName string, options *armnetwork.RouteTablesClientBeginDeleteOptions) error {
	poller, err := c.RouteTablesClient.BeginDelete(ctx, resourceGroupName, routeTableName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
