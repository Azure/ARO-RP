package storage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type BlobContainersClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, containerName string) (mgmtstorage.BlobContainer, error)
	Create(ctx context.Context, resourceGroupName string, accountName string, containerName string, blobContainer mgmtstorage.BlobContainer) (mgmtstorage.BlobContainer, error)
	Delete(ctx context.Context, resourceGroupName string, accountName string, containerName string) (result autorest.Response, err error)
}

type blobContainersClient struct {
	mgmtstorage.BlobContainersClient
}

func NewBlobContainersClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) *blobContainersClient {
	client := mgmtstorage.NewBlobContainersClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	return &blobContainersClient{
		BlobContainersClient: client,
	}
}
