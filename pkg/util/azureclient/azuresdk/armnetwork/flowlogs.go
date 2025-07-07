package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// FlowLogsClientInterface is a minimal interface for azure FlowLogsClientInterface
type FlowLogsClientInterface interface {
	Get(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, options *armnetwork.FlowLogsClientGetOptions) (result armnetwork.FlowLogsClientGetResponse, err error)

	FlowLogsClientAddons
}

type FlowLogsClient struct {
	*armnetwork.FlowLogsClient
}

func NewFlowLogsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (*FlowLogsClient, error) {
	client, err := armnetwork.NewFlowLogsClient(subscriptionID, credential, options)
	return &FlowLogsClient{client}, err
}
