package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

// RouteTablesClientAddons contains addons for RouteTablesClient
type RouteTablesClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string, routeTableName string) error
}

func (c *routeTablesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, routeTableName string) error {
	future, err := c.Delete(ctx, resourceGroupName, routeTableName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
