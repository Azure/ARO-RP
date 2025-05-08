package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// DenyAssignmentClientAddons contains addons for DenyAssignmentClient
type DenyAssignmentClientAddons interface {
	ListForResourceGroup(ctx context.Context, resourceGroupName string, filter string) (result []mgmtauthorization.DenyAssignment, err error)
}

func (c *denyAssignmentClient) ListForResourceGroup(ctx context.Context, resourceGroupName string, filter string) (result []mgmtauthorization.DenyAssignment, err error) {
	page, err := c.DenyAssignmentsClient.ListForResourceGroup(ctx, resourceGroupName, filter)
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
