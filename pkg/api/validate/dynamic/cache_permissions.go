package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

type permissionsCacheClient interface {
	ListForResource(ctx context.Context, resourceGroupName string, resourceProviderNamespace string, parentResourcePath string, resourceType string, resourceName string) (permissions []mgmtauthorization.Permission, err error)
	ListForResourceGroup(ctx context.Context, resourceGroupName string) (permissions []mgmtauthorization.Permission, err error)
}

type permissionsListForResourceCacheKey struct {
	resourceGroupName         string
	resourceProviderNamespace string
	parentResourcePath        string
	resourceType              string
	resourceName              string
}

type permissionsListForResourceGroupCacheKey struct {
	resourceGroupName string
}

type permissionsCache struct {
	c   permissionsCacheClient
	lr  map[permissionsListForResourceCacheKey][]mgmtauthorization.Permission
	lrg map[permissionsListForResourceGroupCacheKey][]mgmtauthorization.Permission
}

func (prm *permissionsCache) ListForResource(ctx context.Context, resourceGroupName string, resourceProviderNamespace string, parentResourcePath string, resourceType string, resourceName string) ([]mgmtauthorization.Permission, error) {
	if _, ok := prm.lr[permissionsListForResourceCacheKey{resourceGroupName, resourceProviderNamespace, parentResourcePath, resourceType, resourceName}]; !ok {
		perm, err := prm.c.ListForResource(ctx, resourceGroupName, resourceProviderNamespace, parentResourcePath, resourceType, resourceName)
		if err != nil {
			return perm, err
		}

		prm.lr[permissionsListForResourceCacheKey{resourceGroupName, resourceProviderNamespace, parentResourcePath, resourceType, resourceName}] = perm
	}

	return prm.lr[permissionsListForResourceCacheKey{resourceGroupName, resourceProviderNamespace, parentResourcePath, resourceType, resourceName}], nil
}

func (prm *permissionsCache) ListForResourceGroup(ctx context.Context, resourceGroupName string) ([]mgmtauthorization.Permission, error) {
	if _, ok := prm.lrg[permissionsListForResourceGroupCacheKey{resourceGroupName}]; !ok {
		perm, err := prm.c.ListForResourceGroup(ctx, resourceGroupName)
		if err != nil {
			return perm, err
		}

		prm.lrg[permissionsListForResourceGroupCacheKey{resourceGroupName}] = perm
	}

	return prm.lrg[permissionsListForResourceGroupCacheKey{resourceGroupName}], nil
}

// newPermissionsCache returns a new permissionsCache. It knows nothing
// about updates and is not thread-safe, but it does retry in the face of
// errors.
func newPermissionsCache(c permissionsCacheClient) permissionsCacheClient {
	return &permissionsCache{
		c:   c,
		lr:  map[permissionsListForResourceCacheKey][]mgmtauthorization.Permission{},
		lrg: map[permissionsListForResourceGroupCacheKey][]mgmtauthorization.Permission{},
	}
}
