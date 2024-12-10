package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

type virtualNetworksGetClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientGetOptions) (vnet armnetwork.VirtualNetworksClientGetResponse, err error)
}

type virtualNetworksCacheKey struct {
	resourceGroupName  string
	virtualNetworkName string
	options            *armnetwork.VirtualNetworksClientGetOptions
}

type virtualNetworksCache struct {
	c virtualNetworksGetClient
	m map[virtualNetworksCacheKey]armnetwork.VirtualNetworksClientGetResponse
}

func (vnc *virtualNetworksCache) Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientGetOptions) (armnetwork.VirtualNetworksClientGetResponse, error) {
	if _, ok := vnc.m[virtualNetworksCacheKey{resourceGroupName, virtualNetworkName, options}]; !ok {
		vnet, err := vnc.c.Get(ctx, resourceGroupName, virtualNetworkName, options)
		if err != nil {
			return vnet, err
		}

		vnc.m[virtualNetworksCacheKey{resourceGroupName, virtualNetworkName, options}] = vnet
	}

	return vnc.m[virtualNetworksCacheKey{resourceGroupName, virtualNetworkName, options}], nil
}

// newVirtualNetworksCache returns a new virtualNetworksCache.  It knows nothing
// about updates and is not thread-safe, but it does retry in the face of
// errors.
func newVirtualNetworksCache(c virtualNetworksGetClient) virtualNetworksGetClient {
	return &virtualNetworksCache{
		c: c,
		m: map[virtualNetworksCacheKey]armnetwork.VirtualNetworksClientGetResponse{},
	}
}
