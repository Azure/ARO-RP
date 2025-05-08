package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
)

// ResourceSkusClientAddons contains addons for ResourceSkusClient
type ResourceSkusClientAddons interface {
	List(ctx context.Context, filter string) (resourceSkus []mgmtcompute.ResourceSku, err error)
}

func (c *resourceSkusClient) List(ctx context.Context, filter string) (resourceSkus []mgmtcompute.ResourceSku, err error) {
	page, err := c.ResourceSkusClient.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		resourceSkus = append(resourceSkus, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return resourceSkus, nil
}
