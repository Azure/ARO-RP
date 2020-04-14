package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

// ResourceGroupsClientAddons contains addons for ResourceGroupsClient
type ResourceGroupsClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string) (err error)
}

func (c *resourceGroupsClient) DeleteAndWait(ctx context.Context, resourceGroupName string) error {
	future, err := c.Delete(ctx, resourceGroupName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
