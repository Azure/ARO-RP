package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

// InterfacesClientAddons contains addons for InterfacesClient
type InterfacesClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string, parameters armnetwork.Interface, options *armnetwork.InterfacesClientBeginCreateOrUpdateOptions) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginDeleteOptions) (err error)
}

func (c *interfacesClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string, parameters armnetwork.Interface, options *armnetwork.InterfacesClientBeginCreateOrUpdateOptions) error {
	poller, err := c.InterfacesClient.BeginCreateOrUpdate(ctx, resourceGroupName, networkInterfaceName, parameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *interfacesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginDeleteOptions) error {
	poller, err := c.InterfacesClient.BeginDelete(ctx, resourceGroupName, networkInterfaceName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
