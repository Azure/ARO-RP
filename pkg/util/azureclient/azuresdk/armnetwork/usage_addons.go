package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

// UsageClientAddons contains addons to UsageClient
type UsageClientAddons interface {
	List(ctx context.Context, location string, options *armnetwork.UsagesClientListOptions) (result []*armnetwork.Usage, err error)
}

func (c *usagesClient) List(ctx context.Context, location string, options *armnetwork.UsagesClientListOptions) (result []*armnetwork.Usage, err error) {
	pager := c.UsagesClient.NewListPager(location, options)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}
	return result, nil
}
