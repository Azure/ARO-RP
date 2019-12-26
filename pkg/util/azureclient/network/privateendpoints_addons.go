package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
)

// PrivateEndpointsClientAddons contains addons for PrivateEndpointsClient
type PrivateEndpointsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, privateEndpointName string, parameters network.PrivateEndpoint) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string) (err error)
}

func (c *privateEndpointsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, privateEndpointName string, parameters network.PrivateEndpoint) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, privateEndpointName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.PrivateEndpointsClient.Client)
}

func (c *privateEndpointsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string) error {
	future, err := c.Delete(ctx, resourceGroupName, publicIPAddressName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.PrivateEndpointsClient.Client)
}
