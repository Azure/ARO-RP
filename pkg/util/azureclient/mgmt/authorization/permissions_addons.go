package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// PermissionsClientAddons contains addons for PermissionsClient
type PermissionsClientAddons interface {
	ListForResource(ctx context.Context, resourceGroupName string, resourceProviderNamespace string, parentResourcePath string, resourceType string, resourceName string) (permissions []mgmtauthorization.Permission, err error)
	ListForResourceGroup(ctx context.Context, resourceGroupName string) (permissions []mgmtauthorization.Permission, err error)
}

func (c *permissionsClient) ListForResource(ctx context.Context, resourceGroupName string, resourceProviderNamespace string, parentResourcePath string, resourceType string, resourceName string) (permissions []mgmtauthorization.Permission, err error) {
	page, err := c.PermissionsClient.ListForResource(ctx, resourceGroupName, resourceProviderNamespace, parentResourcePath, resourceType, resourceName)
	if err != nil {
		return nil, err
	}

	for {
		permissions = append(permissions, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}

		if !page.NotDone() {
			break
		}
	}

	return permissions, nil
}

func (c *permissionsClient) ListForResourceGroup(ctx context.Context, resourceGroupName string) (permissions []mgmtauthorization.Permission, error error) {
	page, err := c.PermissionsClient.ListForResourceGroup(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for {
		permissions = append(permissions, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}

		if !page.NotDone() {
			break
		}
	}

	return permissions, nil
}
