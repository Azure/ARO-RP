package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
)

// ProvidersClientAddons is a minimal interface for azure ProvidersClient
type ProvidersClientAddons interface {
	List(ctx context.Context, top *int32, expand string) (providers []mgmtfeatures.Provider, err error)
}

func (c *providersClient) List(ctx context.Context, top *int32, expand string) (providers []mgmtfeatures.Provider, err error) {
	page, err := c.ProvidersClient.List(ctx, top, expand)
	if err != nil {
		return nil, err
	}
	for page.NotDone() {
		providers = append(providers, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return providers, nil
}
