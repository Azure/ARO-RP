package resources

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
)

// ProvidersClientAddons is a minimal interface for azure ProvidersClient
type ProvidersClientAddons interface {
	List(ctx context.Context, top *int32, expand string) (providers []mgmtresources.Provider, err error)
}

func (c *providersClient) List(ctx context.Context, top *int32, expand string) (providers []mgmtresources.Provider, err error) {
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
