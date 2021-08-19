package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

type virtualNetworksGetClient interface {
	Get(context.Context, string, string, string) (mgmtnetwork.VirtualNetwork, error)
}

type virtualNetworksCacheKey struct {
	resourceGroupName  string
	virtualNetworkName string
	expand             string
}

type virtualNetworksCache struct {
	c virtualNetworksGetClient
	m map[virtualNetworksCacheKey]mgmtnetwork.VirtualNetwork
}

func (vnc *virtualNetworksCache) Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, expand string) (mgmtnetwork.VirtualNetwork, error) {
	if _, ok := vnc.m[virtualNetworksCacheKey{resourceGroupName, virtualNetworkName, expand}]; !ok {
		vnet, err := vnc.c.Get(ctx, resourceGroupName, virtualNetworkName, expand)
		if err != nil {
			return vnet, err
		}

		vnc.m[virtualNetworksCacheKey{resourceGroupName, virtualNetworkName, expand}] = vnet
	}

	return vnc.m[virtualNetworksCacheKey{resourceGroupName, virtualNetworkName, expand}], nil
}

// newVirtualNetworksCache returns a new virtualNetworksCache.  It knows nothing
// about updates and is not thread-safe, but it does retry in the face of
// errors.
func newVirtualNetworksCache(c virtualNetworksGetClient) virtualNetworksGetClient {
	return &virtualNetworksCache{
		c: c,
		m: map[virtualNetworksCacheKey]mgmtnetwork.VirtualNetwork{},
	}
}
