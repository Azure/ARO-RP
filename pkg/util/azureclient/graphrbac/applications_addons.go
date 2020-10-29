package graphrbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
)

// ApplicationsClientAddons is a minimal interface for azure ApplicationsClient
type ApplicationsClientAddons interface {
	List(ctx context.Context, filter string) (result []graphrbac.Application, err error)
}

func (sc *applicationsClient) List(ctx context.Context, filter string) (result []graphrbac.Application, err error) {
	page, err := sc.ApplicationsClient.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		result = append(result, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
