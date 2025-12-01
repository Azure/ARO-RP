package azblob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

// BlobsClient is a minimal interface for Azure BlobsClient
type BlobsClient interface {
	DownloadStream(ctx context.Context, containerName string, blobName string, o *azblob.DownloadStreamOptions) (azblob.DownloadStreamResponse, error)
	UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte, o *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error)
	DeleteBlob(ctx context.Context, containerName string, blobName string, o *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error)
	ServiceClient() *service.Client
	BlobsClientAddons
}

type blobsClient struct {
	*azblob.Client
}

var _ BlobsClient = &blobsClient{}

// NewBlobsClientUsingSAS creates a new BlobsClient using SAS
func NewBlobsClientUsingSAS(sasURL string, options *arm.ClientOptions) (*blobsClient, error) {
	azBlobOptions := &azblob.ClientOptions{
		ClientOptions: (*options).ClientOptions,
	}
	client, err := azblob.NewClientWithNoCredential(sasURL, azBlobOptions)
	if err != nil {
		return nil, err
	}

	return &blobsClient{
		Client: client,
	}, nil
}

// NewBlobsClientUsingEntra creates a new BlobsClient Microsoft Entra credentials
func NewBlobsClientUsingEntra(serviceURL string, credential azcore.TokenCredential, options *arm.ClientOptions) (*blobsClient, error) {
	azBlobOptions := &azblob.ClientOptions{
		ClientOptions: (*options).ClientOptions,
	}
	client, err := azblob.NewClient(serviceURL, credential, azBlobOptions)
	if err != nil {
		return nil, err
	}

	return &blobsClient{
		Client: client,
	}, nil
}
