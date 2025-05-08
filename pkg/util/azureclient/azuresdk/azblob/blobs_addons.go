package azblob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
)

type BlobsClientAddons interface {
	BlobExists(ctx context.Context, container string, blobPath string) (bool, error)
	DeleteContainer(ctx context.Context, container string) error
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
