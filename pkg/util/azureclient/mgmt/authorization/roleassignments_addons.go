package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// RoleAssignmentsClientAddons contains addons for RoleAssignmentsClient
type RoleAssignmentsClientAddons interface {
	ListForResource(ctx context.Context, resourceGroupName string, resourceProviderNamespace string, parentResourcePath string, resourceType string, resourceName string, filter string) ([]mgmtauthorization.RoleAssignment, error)
	ListForResourceGroup(ctx context.Context, resourceGroupName string, filter string) ([]mgmtauthorization.RoleAssignment, error)
}

func (c *roleAssignmentsClient) ListForResource(ctx context.Context, resourceGroupName string, resourceProviderNamespace string, parentResourcePath string, resourceType string, resourceName string, filter string) (result []mgmtauthorization.RoleAssignment, err error) {
	page, err := c.RoleAssignmentsClient.ListForResource(ctx, resourceGroupName, resourceProviderNamespace, parentResourcePath, resourceType, resourceName, filter)
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

func (c *roleAssignmentsClient) ListForResourceGroup(ctx context.Context, resourceGroupName string, filter string) (result []mgmtauthorization.RoleAssignment, err error) {
	page, err := c.RoleAssignmentsClient.ListForResourceGroup(ctx, resourceGroupName, filter)
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
