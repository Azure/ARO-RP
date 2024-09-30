package azblob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// BlobsClient is a minimal interface for Azure BlobsClient
type BlobsClient interface {
	DownloadStream(ctx context.Context, containerName string, blobName string, o *azblob.DownloadStreamOptions) (azblob.DownloadStreamResponse, error)
	UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte, o *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error)
	DeleteBlob(ctx context.Context, containerName string, blobName string, o *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error)
	BlobExists(ctx context.Context, container string, blobPath string) (bool, error)
	DeleteContainer(ctx context.Context, container string) error
}

type blobsClient struct {
	*azblob.Client
}

var _ BlobsClient = &blobsClient{}

// NewBlobsClient creates a new BlobsClient using SAS
func NewBlobsClientUsingSAS(ctx context.Context, environment *azureclient.AROEnvironment, sasURL string) (*blobsClient, error) {
	customRoundTripper := azureclient.NewCustomRoundTripper(http.DefaultTransport)

	options := &azblob.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
			Transport: &http.Client{
				Transport: customRoundTripper,
			},
		},
	}
	client, err := azblob.NewClientWithNoCredential(sasURL, options)
	if err != nil {
		return nil, err
	}

	return &blobsClient{
		Client: client,
	}, nil
}

func NewBlobsClient(ctx context.Context, environment *azureclient.AROEnvironment, serviceURL string, credential azcore.TokenCredential) (*blobsClient, error) {
	customRoundTripper := azureclient.NewCustomRoundTripper(http.DefaultTransport)

	options := &azblob.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
			Transport: &http.Client{
				Transport: customRoundTripper,
			},
		},
	}
	client, err := azblob.NewClient(serviceURL, credential, options)
	if err != nil {
		return nil, err
	}

	return &blobsClient{
		Client: client,
	}, nil
}

func (client *blobsClient) BlobExists(ctx context.Context, container string, blobPath string) (bool, error) {
	blobRef := client.ServiceClient().NewContainerClient(container).NewBlobClient(blobPath)
	_, err := blobRef.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(
			err,
			bloberror.BlobNotFound,
			bloberror.ContainerNotFound,
			bloberror.ResourceNotFound,
			bloberror.CannotVerifyCopySource,
		) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func (client *blobsClient) DeleteContainer(ctx context.Context, container string) error {
	containerRef := client.ServiceClient().NewContainerClient(container)
	_, err := containerRef.Delete(ctx, nil)
	if err != nil {
		if bloberror.HasCode(
			err,
			bloberror.ContainerNotFound,
			bloberror.ResourceNotFound,
		) {
			return nil
		}
	}
	return err
}
