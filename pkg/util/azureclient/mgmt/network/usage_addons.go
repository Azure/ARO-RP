package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
)

// UsageClientAddons contains addons to UsageClient
type UsageClientAddons interface {
	List(ctx context.Context, location string) (result []mgmtnetwork.Usage, err error)
}

func (u *usageClient) List(ctx context.Context, location string) (result []mgmtnetwork.Usage, err error) {
	page, err := u.UsagesClient.List(ctx, location)
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
