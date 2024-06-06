package azblob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	azstorage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armstorage"
)

type Manager interface {
	CreateBlobContainer(ctx context.Context, resourceGroup string, account string, container string, publicAccess azstorage.PublicAccess) error
	DeleteBlobContainer(ctx context.Context, resourceGroupName string, accountName string, containerName string) error
	GetAZBlobClient(blobContainerURL string, options *azblob.ClientOptions) (AZBlobClient, error)
}

type manager struct {
	cred          azcore.TokenCredential
	blobContainer armstorage.BlobContainersClient
}

func NewManager(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (Manager, error) {
	client, err := armstorage.NewBlobContainersClient(environment, subscriptionID, credential)
	if err != nil {
		return nil, err
	}
	return &manager{
		cred:          credential,
		blobContainer: client,
	}, nil
}

func (m *manager) CreateBlobContainer(ctx context.Context, resourceGroup string, accountName string, containerName string, publicAccess azstorage.PublicAccess) error {
	needToCreateBlobContainer := false

	_, err := m.blobContainer.Get(
		ctx,
		resourceGroup,
		accountName,
		containerName,
		&azstorage.BlobContainersClientGetOptions{},
	)
	if err != nil {
		if !bloberror.HasCode(err, bloberror.ContainerNotFound) {
			return err
		}
		needToCreateBlobContainer = true
	}

	if !needToCreateBlobContainer {
		return nil
	}

	_, err = m.blobContainer.Create(
		ctx,
		resourceGroup,
		accountName,
		containerName,
		azstorage.BlobContainer{
			ContainerProperties: &azstorage.ContainerProperties{
				PublicAccess: to.Ptr(publicAccess),
			},
		},
		&azstorage.BlobContainersClientCreateOptions{},
	)

	return err
}

func (m *manager) DeleteBlobContainer(ctx context.Context, resourceGroupName string, accountName string, containerName string) error {
	_, err := m.blobContainer.Get(
		ctx,
		resourceGroupName,
		accountName,
		containerName,
		&azstorage.BlobContainersClientGetOptions{},
	)
	if err != nil {
		if bloberror.HasCode(err, bloberror.ContainerNotFound) {
			return nil
		}
	}

	_, err = m.blobContainer.Delete(
		ctx,
		resourceGroupName,
		accountName,
		containerName,
		&azstorage.BlobContainersClientDeleteOptions{},
	)
	return err
}

func (m *manager) GetAZBlobClient(blobContainerURL string, options *azblob.ClientOptions) (AZBlobClient, error) {
	return NewAZBlobClient(blobContainerURL, m.cred, options)
}

type AZBlobClient interface {
	UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte) error
}

type azBlobClient struct {
	client *azblob.Client
}

func NewAZBlobClient(blobContainerURL string, credential azcore.TokenCredential, options *azblob.ClientOptions) (AZBlobClient, error) {
	client, err := azblob.NewClient(blobContainerURL, credential, options)
	if err != nil {
		return nil, err
	}
	return &azBlobClient{client: client}, nil
}

func (azBlobClient *azBlobClient) UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte) error {
	_, err := azBlobClient.client.UploadBuffer(ctx, containerName, blobName, buffer, &azblob.UploadBufferOptions{})
	return err
}
