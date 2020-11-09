package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// RoleAssignmentsClientAddons contains addons for RoleAssignmentsClient
type RoleAssignmentsClientAddons interface {
	List(ctx context.Context, filter string) (result []mgmtauthorization.RoleAssignment, err error)
}

func (c *roleAssignmentsClient) List(ctx context.Context, filter string) (result []mgmtauthorization.RoleAssignment, err error) {
	page, err := c.RoleAssignmentsClient.List(ctx, filter)
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
