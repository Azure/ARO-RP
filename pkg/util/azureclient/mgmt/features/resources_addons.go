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
	UpdateByIDAndWait(ctx context.Context, resourceID string, APIVersion string, parameters mgmtfeatures.GenericResource) error
	ListByResourceGroup(ctx context.Context, resourceGroupName string, filter string, expand string, top *int32) ([]mgmtfeatures.GenericResourceExpanded, error)
}

func (c *resourcesClient) Client() autorest.Client {
	return c.ResourcesClient.Client
}

func (c *resourcesClient) UpdateByIDAndWait(ctx context.Context, resourceID string, APIVersion string, parameters mgmtfeatures.GenericResource) error {
	future, err := c.ResourcesClient.UpdateByID(ctx, resourceID, APIVersion, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client())
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
