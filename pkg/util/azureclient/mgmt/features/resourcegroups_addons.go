package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
)

// ResourceGroupsClientAddons contains addons for ResourceGroupsClient
type ResourceGroupsClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string) (err error)
	List(ctx context.Context, filter string, top *int32) (resourcegroups []mgmtfeatures.ResourceGroup, err error)
}

func (c *resourceGroupsClient) DeleteAndWait(ctx context.Context, resourceGroupName string) error {
	future, err := c.Delete(ctx, resourceGroupName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *resourceGroupsClient) List(ctx context.Context, filter string, top *int32) (resourcegroups []mgmtfeatures.ResourceGroup, err error) {
	page, err := c.ResourceGroupsClient.List(ctx, filter, top)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		resourcegroups = append(resourcegroups, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return resourcegroups, nil
}
