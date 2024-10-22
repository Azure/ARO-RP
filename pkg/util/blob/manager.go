package blob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azstorage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armstorage"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azblob"
)

type Manager interface {
	GetContainerProperties(ctx context.Context, resourceGroupName string, accountName string, containerName string) (azstorage.AccountsClientGetPropertiesResponse, error)
	GetBlobsClient(blobContainerURL string) (azblob.BlobsClient, error)
}

type manager struct {
	cred          azcore.TokenCredential
	account       armstorage.AccountsClient
	clientOptions *arm.ClientOptions
}

func NewManager(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (Manager, error) {
	accountsClient, err := armstorage.NewAccountsClient(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &manager{
		cred:          credential,
		account:       accountsClient,
		clientOptions: options,
	}, nil
}

func (m *manager) GetContainerProperties(ctx context.Context, resourceGroupName string, accountName string, containerName string) (azstorage.AccountsClientGetPropertiesResponse, error) {
	return m.account.GetProperties(ctx, resourceGroupName, accountName, &azstorage.AccountsClientGetPropertiesOptions{})
}

func (m *manager) GetBlobsClient(blobContainerURL string) (azblob.BlobsClient, error) {
	return azblob.NewBlobsClientUsingEntra(blobContainerURL, m.cred, m.clientOptions)
}
