package armcontainerinstance

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance"
)

// ContainerGroupsClientAddons contains addons for ContainerGroupsClient.
type ContainerGroupsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, containerGroupName string, containerGroup armcontainerinstance.ContainerGroup, options *armcontainerinstance.ContainerGroupsClientBeginCreateOrUpdateOptions) (armcontainerinstance.ContainerGroupsClientCreateOrUpdateResponse, error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, containerGroupName string, options *armcontainerinstance.ContainerGroupsClientBeginDeleteOptions) error
}

func (c *containerGroupsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, containerGroupName string, containerGroup armcontainerinstance.ContainerGroup, options *armcontainerinstance.ContainerGroupsClientBeginCreateOrUpdateOptions) (armcontainerinstance.ContainerGroupsClientCreateOrUpdateResponse, error) {
	poller, err := c.BeginCreateOrUpdate(ctx, resourceGroupName, containerGroupName, containerGroup, options)
	if err != nil {
		return armcontainerinstance.ContainerGroupsClientCreateOrUpdateResponse{}, err
	}
	return poller.PollUntilDone(ctx, nil)
}

func (c *containerGroupsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, containerGroupName string, options *armcontainerinstance.ContainerGroupsClientBeginDeleteOptions) error {
	poller, err := c.BeginDelete(ctx, resourceGroupName, containerGroupName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
