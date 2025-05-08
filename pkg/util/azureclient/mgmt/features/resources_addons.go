package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
)

// ResourcesClientAddons is a minimal interface for azure ResourcesClient
type ResourcesClientAddons interface {
	Client() autorest.Client
	ListByResourceGroup(ctx context.Context, resourceGroupName string, filter string, expand string, top *int32) ([]mgmtfeatures.GenericResourceExpanded, error)
	DeleteByIDAndWait(ctx context.Context, resourceID string, apiVersion string) error
}

func (c *resourcesClient) Client() autorest.Client {
	return c.ResourcesClient.Client
}

func (c *resourcesClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, filter string, expand string, top *int32) (resources []mgmtfeatures.GenericResourceExpanded, err error) {
	page, err := c.ResourcesClient.ListByResourceGroup(ctx, resourceGroupName, filter, expand, top)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		resources = append(resources, page.Values()...)
		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return resources, nil
}

func (c *resourcesClient) DeleteByIDAndWait(ctx context.Context, resourceID string, apiVersion string) error {
	future, err := c.DeleteByID(ctx, resourceID, apiVersion)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client())
}
