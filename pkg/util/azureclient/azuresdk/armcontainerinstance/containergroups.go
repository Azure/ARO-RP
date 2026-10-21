package armcontainerinstance

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// ContainerGroupsClient is a minimal interface for azure ContainerGroupsClient.
type ContainerGroupsClient interface {
	Get(ctx context.Context, resourceGroupName string, containerGroupName string, options *armcontainerinstance.ContainerGroupsClientGetOptions) (armcontainerinstance.ContainerGroupsClientGetResponse, error)
	ContainerGroupsClientAddons
}

type containerGroupsClient struct {
	*armcontainerinstance.ContainerGroupsClient
}

var _ ContainerGroupsClient = &containerGroupsClient{}

// NewContainerGroupsClient creates a new ContainerGroupsClient.
func NewContainerGroupsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (ContainerGroupsClient, error) {
	client, err := armcontainerinstance.NewContainerGroupsClient(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &containerGroupsClient{ContainerGroupsClient: client}, nil
}
