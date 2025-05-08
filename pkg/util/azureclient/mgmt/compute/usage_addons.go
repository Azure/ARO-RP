package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
)

// UsageClientAddons contains addons to UsageClient
type UsageClientAddons interface {
	List(ctx context.Context, location string) (result []mgmtcompute.Usage, err error)
}

func (u *usageClient) List(ctx context.Context, location string) (result []mgmtcompute.Usage, err error) {
	page, err := u.UsageClient.List(ctx, location)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		result = append(result, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
