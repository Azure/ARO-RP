package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
)

// ResourcesClientAddons is a minimal interface for azure ResourcesClient
type ResourcesClientAddons interface {
	List(ctx context.Context, filter, expand string, top *int32) ([]mgmtfeatures.GenericResourceExpanded, error)
}

func (c *resourcesClient) List(ctx context.Context, filter, expand string, top *int32) (resources []mgmtfeatures.GenericResourceExpanded, err error) {
	page, err := c.ResourcesClient.List(ctx, filter, expand, top)
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
