package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

// InterfacesClientAddons contains addons for InterfacesClient
type InterfacesClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string, parameters mgmtnetwork.Interface) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string) (err error)
}

func (c *interfacesClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string, parameters mgmtnetwork.Interface) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, networkInterfaceName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *interfacesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string) error {
	future, err := c.Delete(ctx, resourceGroupName, networkInterfaceName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
