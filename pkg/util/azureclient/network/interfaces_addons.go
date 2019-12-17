package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

// InterfacesClientAddons contains addons for InterfacesClient
type InterfacesClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string) (err error)
}

func (c *interfacesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string) error {
	future, err := c.Delete(ctx, resourceGroupName, networkInterfaceName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.InterfacesClient.Client)
}
