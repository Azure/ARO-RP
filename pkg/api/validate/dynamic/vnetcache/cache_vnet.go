package vnetcache

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/azure"

	networkutil "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
)

type VirtualNetworksGetClient interface {
	Get(ctx context.Context, key VirtualNetworksCacheKey) (mgmtnetwork.VirtualNetwork, error)
}

type VirtualNetworksCacheKey struct {
	ResourceGroupName  string
	VirtualNetworkName string
}

type virtualNetworksCache struct {
	c networkutil.VirtualNetworksClient
	m map[VirtualNetworksCacheKey]mgmtnetwork.VirtualNetwork
}

func CacheKeyFromResource(resource azure.Resource) VirtualNetworksCacheKey {
	return VirtualNetworksCacheKey{ResourceGroupName: resource.ResourceGroup, VirtualNetworkName: resource.ResourceName}
}

func (vnc *virtualNetworksCache) Get(ctx context.Context, key VirtualNetworksCacheKey) (mgmtnetwork.VirtualNetwork, error) {
	if _, ok := vnc.m[key]; !ok {
		vnet, err := vnc.c.Get(ctx, key.ResourceGroupName, key.VirtualNetworkName, "")
		if err != nil {
			return vnet, err
		}

		vnc.m[key] = vnet
	}

	return vnc.m[key], nil
}

// newVirtualNetworksCache returns a new virtualNetworksCache.  It knows nothing
// about updates and is not thread-safe, but it does retry in the face of
// errors.
func NewVirtualNetworksCache(c networkutil.VirtualNetworksClient) VirtualNetworksGetClient {
	return &virtualNetworksCache{
		c: c,
		m: map[VirtualNetworksCacheKey]mgmtnetwork.VirtualNetwork{},
	}
}
