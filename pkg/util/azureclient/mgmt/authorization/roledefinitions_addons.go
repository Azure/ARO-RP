package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// RoleDefinitionsClientAddons contains addons for RoleDefinitionsClient
type RoleDefinitionsClientAddons interface {
	List(ctx context.Context, scope string, filter string) ([]mgmtauthorization.RoleDefinition, error)
}

func (c *roleDefinitionsClient) List(ctx context.Context, scope string, filter string) (result []mgmtauthorization.RoleDefinition, err error) {
	page, err := c.RoleDefinitionsClient.List(ctx, scope, filter)
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
