package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

// FlowLogsClientAddons contains addons to WatchersClient
type FlowLogsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, parameters mgmtnetwork.FlowLog) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string) error
}

func (c *flowLogsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, parameters mgmtnetwork.FlowLog) error {
	future, err := c.FlowLogsClient.CreateOrUpdate(ctx, resourceGroupName, networkWatcherName, flowLogName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *flowLogsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string) error {
	future, err := c.FlowLogsClient.Delete(ctx, resourceGroupName, networkWatcherName, flowLogName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
