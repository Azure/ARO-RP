package storage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armstorage"
)

type Manager interface {
	BlobService(account string, containerName string) (*container.Client, error)
}

type manager struct {
	env            *azureclient.AROEnvironment
	cred           azcore.TokenCredential
	blobContainers armstorage.BlobContainersClient
}

func NewManager(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (Manager, error) {
	blobContainers, err := armstorage.NewBlobContainersClient(environment, subscriptionID, credential)
	if err != nil {
		return nil, err
	}

	return &manager{
		env:            environment,
		cred:           credential,
		blobContainers: blobContainers,
	}, nil
}

func (m *manager) BlobService(account string, containerName string) (*container.Client, error) {
	containerURL := fmt.Sprintf(
		"https://%s.blob.%s/%s",
		account,
		m.env.StorageEndpointSuffix,
		containerName,
	)

	options := container.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: m.env.Cloud,
		},
	}

	client, err := container.NewClient(containerURL, m.cred, &options)
	if err != nil {
		return nil, err
	}

	return client, nil
}
