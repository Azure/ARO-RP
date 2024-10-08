package azblob2

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/sirupsen/logrus"
)

type AZBlobClient interface {
	DownloadStream(ctx context.Context, containerName string, blobName string, o *azblob.DownloadStreamOptions) ([]byte, error)
	UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte) error
	DeleteBlob(ctx context.Context, containerName string, blobName string) error
	Exists(ctx context.Context, container string, blobPath string) (bool, error)
}

type azBlobClient struct {
	client *azblob.Client
}

func NewAZBlobClient(ctx context.Context, blobContainerURL string, credential azcore.TokenCredential, options *azblob.ClientOptions, isUserDesignated bool, log *logrus.Entry) (AZBlobClient, error) {
	client, err := azblob.NewClient(blobContainerURL, credential, options)
	if err != nil {
		return nil, err
	}
	blobClient := &azBlobClient{client: client}
	if isUserDesignated {
		sasURL, err := blobClient.signBlobURL(ctx, blobContainerURL, time.Now().UTC().Add(2*time.Hour))
		log.Printf("sasURL -------- %s", sasURL)
		if err != nil {
			return nil, err
		}

		client, err = azblob.NewClientWithNoCredential(sasURL, options)
		if err != nil {
			return nil, err
		}
		blobClient = &azBlobClient{client: client}
	}
	return blobClient, nil
}

func (azBlobClient *azBlobClient) DownloadStream(ctx context.Context, containerName string, blobName string, o *azblob.DownloadStreamOptions) ([]byte, error) {
	response, err := azBlobClient.client.DownloadStream(ctx, containerName, blobName, o)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

func (azBlobClient *azBlobClient) UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte) error {
	_, err := azBlobClient.client.UploadBuffer(ctx, containerName, blobName, buffer, &azblob.UploadBufferOptions{})
	return err
}

func (azBlobClient *azBlobClient) DeleteBlob(ctx context.Context, containerName string, blobName string) error {
	_, err := azBlobClient.client.DeleteBlob(ctx, containerName, blobName, &azblob.DeleteBlobOptions{})
	return err
}

func (azBlobClient *azBlobClient) signBlobURL(ctx context.Context, blobURL string, expires time.Time) (string, error) {
	urlParts, err := sas.ParseURL(blobURL)
	if err != nil {
		return "", err
	}
	// perms := sas.BlobPermissions{Read: true, Write: true, Create: true}
	perms := sas.BlobPermissions{Read: true, Write: true, Create: true}
	signatureValues := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().UTC().Add(-10 * time.Second),
		ExpiryTime:    expires,
		Permissions:   perms.String(),
		ContainerName: "aro",
		BlobName:      "graph",
	}
	urlParts.SAS, err = azBlobClient.sign(ctx, &signatureValues)
	if err != nil {
		return "", err
	}
	return urlParts.String(), nil
}

func (azBlobClient *azBlobClient) sign(ctx context.Context, signatureValues *sas.BlobSignatureValues) (sas.QueryParameters, error) {
	currentTime := time.Now().UTC().Add(-10 * time.Second)
	expiryTime := currentTime.Add(2 * time.Hour)

	info := service.KeyInfo{
		Start:  to.Ptr(currentTime.UTC().Format(sas.TimeFormat)),
		Expiry: to.Ptr(expiryTime.UTC().Format(sas.TimeFormat)),
	}

	udc, err := azBlobClient.client.ServiceClient().GetUserDelegationCredential(ctx, info, nil)
	if err != nil {
		return sas.QueryParameters{}, err
	}
	return signatureValues.SignWithUserDelegation(udc)
}

func (azBlobClient *azBlobClient) Exists(ctx context.Context, container string, blobPath string) (bool, error) {
	blobRef := azBlobClient.client.ServiceClient().NewContainerClient(container).NewBlobClient(blobPath)
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
