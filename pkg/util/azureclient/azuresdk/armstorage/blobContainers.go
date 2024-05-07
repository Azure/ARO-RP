package armstorage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// BlobContainersClient is a minimal interface for Azure BlobContainersClient
type BlobContainersClient interface {
	Create(ctx context.Context, resourceGroupName string, accountName string, containerName string, blobContainer armstorage.BlobContainer, options *armstorage.BlobContainersClientCreateOptions) (armstorage.BlobContainersClientCreateResponse, error)
	Get(ctx context.Context, resourceGroupName string, accountName string, containerName string, options *armstorage.BlobContainersClientGetOptions) (armstorage.BlobContainersClientGetResponse, error)
	Delete(ctx context.Context, resourceGroupName string, accountName string, containerName string, options *armstorage.BlobContainersClientDeleteOptions) (armstorage.BlobContainersClientDeleteResponse, error)
}

type blobContainersClient struct {
	*armstorage.BlobContainersClient
}

var _ BlobContainersClient = &blobContainersClient{}

// NewBlobContainersClient creates a new BlobContainersClient
func NewBlobContainersClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (BlobContainersClient, error) {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	clientFactory, err := armstorage.NewClientFactory(subscriptionID, credential, &options)
	if err != nil {
		return nil, err
	}
	return &blobContainersClient{BlobContainersClient: clientFactory.NewBlobContainersClient()}, nil
}
