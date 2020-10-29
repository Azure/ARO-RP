package graphrbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
)

// ServicePrincipalClientAddons is a minimal interface for azure ServicePrincipalClient
type ServicePrincipalClientAddons interface {
	List(ctx context.Context, filter string) (result []graphrbac.ServicePrincipal, err error)
}

func (sc *servicePrincipalClient) List(ctx context.Context, filter string) (result []graphrbac.ServicePrincipal, err error) {
	page, err := sc.ServicePrincipalsClient.List(ctx, filter)
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
