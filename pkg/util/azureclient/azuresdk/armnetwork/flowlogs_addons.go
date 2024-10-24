package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// FlowLogsClientAddons contains addons to WatchersClient
type FlowLogsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, parameters armnetwork.FlowLog, options *armnetwork.FlowLogsClientBeginCreateOrUpdateOptions) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, options *armnetwork.FlowLogsClientBeginDeleteOptions) error
}

func (c *flowLogsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, parameters armnetwork.FlowLog, options *armnetwork.FlowLogsClientBeginCreateOrUpdateOptions) error {
	poller, err := c.FlowLogsClient.BeginCreateOrUpdate(ctx, resourceGroupName, networkWatcherName, flowLogName, parameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *flowLogsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, options *armnetwork.FlowLogsClientBeginDeleteOptions) error {
	poller, err := c.FlowLogsClient.BeginDelete(ctx, resourceGroupName, networkWatcherName, flowLogName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
