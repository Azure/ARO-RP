package armcontainerinstance

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// ContainersClient is a minimal interface for azure ContainersClient.
type ContainersClient interface {
	ListLogs(ctx context.Context, resourceGroupName string, containerGroupName string, containerName string, options *armcontainerinstance.ContainersClientListLogsOptions) (armcontainerinstance.ContainersClientListLogsResponse, error)
}

type containersClient struct {
	*armcontainerinstance.ContainersClient
}

var _ ContainersClient = &containersClient{}

// NewContainersClient creates a new ContainersClient.
func NewContainersClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (ContainersClient, error) {
	client, err := armcontainerinstance.NewContainersClient(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &containersClient{ContainersClient: client}, nil
}
