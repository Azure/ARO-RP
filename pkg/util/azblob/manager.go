package azblob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azstorage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armstorage"
)

type Manager interface {
	GetContainerProperties(ctx context.Context, resourceGroupName string, accountName string, containerName string) (azstorage.AccountsClientGetPropertiesResponse, error)
	GetAZBlobClient(blobContainerURL string, options *azblob.ClientOptions) (AZBlobClient, error)
}

type manager struct {
	cred    azcore.TokenCredential
	account armstorage.AccountsClient
}

func NewManager(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (Manager, error) {
	accountsClient, err := armstorage.NewAccountsClient(environment, subscriptionID, credential)
	if err != nil {
		return nil, err
	}
	return &manager{
		cred:    credential,
		account: accountsClient,
	}, nil
}

func (m *manager) GetContainerProperties(ctx context.Context, resourceGroupName string, accountName string, containerName string) (azstorage.AccountsClientGetPropertiesResponse, error) {
	return m.account.GetProperties(ctx, resourceGroupName, accountName, &azstorage.AccountsClientGetPropertiesOptions{})
}

func (m *manager) GetAZBlobClient(blobContainerURL string, options *azblob.ClientOptions) (AZBlobClient, error) {
	return NewAZBlobClient(blobContainerURL, m.cred, options)
}

type AZBlobClient interface {
	UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte) error
	DeleteBlob(ctx context.Context, containerName string, blobName string) error
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

func (azBlobClient *azBlobClient) DeleteBlob(ctx context.Context, containerName string, blobName string) error {
	_, err := azBlobClient.client.DeleteBlob(ctx, containerName, blobName, &azblob.DeleteBlobOptions{})
	return err
}
