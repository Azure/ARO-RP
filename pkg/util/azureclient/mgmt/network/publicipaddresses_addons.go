package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

// PublicIPAddressesClientAddons contains addons for PublicIPAddressesClient
type PublicIPAddressesClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string, parameters mgmtnetwork.PublicIPAddress) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string) (err error)
}

func (c *publicIPAddressesClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string, parameters mgmtnetwork.PublicIPAddress) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, publicIPAddressName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *publicIPAddressesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string) error {
	future, err := c.Delete(ctx, resourceGroupName, publicIPAddressName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
