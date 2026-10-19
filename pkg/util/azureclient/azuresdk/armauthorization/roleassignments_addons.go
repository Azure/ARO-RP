package armauthorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

// RoleAssignmentsClientAddons contains addons for RoleAssignmentsClient
type RoleAssignmentsClientAddons interface {
	ListForResourceGroup(ctx context.Context, resourceGroupName string, options *armauthorization.RoleAssignmentsClientListForResourceGroupOptions) ([]*armauthorization.RoleAssignment, error)
}

func (c *roleAssignmentsClient) ListForResourceGroup(ctx context.Context, resourceGroupName string, options *armauthorization.RoleAssignmentsClientListForResourceGroupOptions) (result []*armauthorization.RoleAssignment, err error) {
	pager := c.NewListForResourceGroupPager(resourceGroupName, options)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}

	return result, nil
}
