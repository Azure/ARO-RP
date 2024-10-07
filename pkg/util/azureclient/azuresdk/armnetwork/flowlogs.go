package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// FlowLogsClient is a minimal interface for azure FlowLogsClient
type FlowLogsClient interface {
	Get(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, options *armnetwork.FlowLogsClientGetOptions) (result armnetwork.FlowLogsClientGetResponse, err error)

	FlowLogsClientAddons
}

type flowLogsClient struct {
	*armnetwork.FlowLogsClient
}

// NewFlowLogsClient creates a new FlowLogsClient
func NewFlowLogsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (FlowLogsClient, error) {
	client, err := armnetwork.NewFlowLogsClient(subscriptionID, credential, options)
	return &flowLogsClient{client}, err
}
