package oidcbuilder

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azblob"
)

const (
	DiscoveryDocumentKey = ".well-known/openid-configuration"
	JWKSKey              = "openid/v1/jwks"
	WebContainer         = "$web"
)

type OIDCBuilder struct {
	privateKey       []byte
	publicKey        []byte
	blobContainerURL string
	endpointURL      string
	directory        string
}

func NewOIDCBuilder(env env.Interface, oidcEndpoint string, directoryName string) (*OIDCBuilder, error) {
	privateKey, publicKey, err := CreateKeyPair(env)
	if err != nil {
		return nil, err
	}

	return &OIDCBuilder{
		privateKey:       privateKey,
		publicKey:        publicKey,
		blobContainerURL: GenerateBlobContainerURL(env),
		endpointURL:      fmt.Sprintf("%s%s", oidcEndpoint, directoryName),
		directory:        directoryName,
	}, nil
}

func GenerateBlobContainerURL(env env.Interface) string {
	return fmt.Sprintf("https://%s.blob.%s/%s", env.OIDCStorageAccountName(), env.Environment().StorageEndpointSuffix, WebContainer)
}

func (b *OIDCBuilder) EnsureOIDCDocs(ctx context.Context, blobsClient azblob.BlobsClient) error {
	// Create the OIDC configuration
	discoveryDocument := GenerateDiscoveryDocument(b.endpointURL)

	// Create the OIDC key list
	jwks, err := BuildJSONWebKeySet(b.publicKey)
	if err != nil {
		return err
	}

	return populateOidcFolder(ctx, b.directory, discoveryDocument, jwks, blobsClient)
}

func (b *OIDCBuilder) GetEndpointUrl() string {
	return b.endpointURL
}

func (b *OIDCBuilder) GetPrivateKey() string {
	return string(b.privateKey)
}

func (b *OIDCBuilder) GetBlobContainerURL() string {
	return b.blobContainerURL
}

func populateOidcFolder(ctx context.Context, directory string, discoveryDocument string, jwks []byte, blobsClient azblob.BlobsClient) error {
	_, err := blobsClient.UploadBuffer(
		ctx,
		"",
		DocumentKey(directory, DiscoveryDocumentKey),
		[]byte(discoveryDocument),
		nil,
	)
	if err != nil {
		return err
	}

	_, err = blobsClient.UploadBuffer(
		ctx,
		"",
		DocumentKey(directory, JWKSKey),
		jwks,
		nil,
	)
	return err
}

func DeleteOidcFolder(ctx context.Context, directory string, blobsClient azblob.BlobsClient) error {
	for _, key := range []string{DiscoveryDocumentKey, JWKSKey} {
		_, err := blobsClient.DeleteBlob(ctx, "", DocumentKey(directory, key), nil)
		if err != nil && !bloberror.HasCode(err, bloberror.BlobNotFound) {
			return err
		}
	}
	return nil
}

func DocumentKey(directory string, blobKey string) string {
	return fmt.Sprintf("%s/%s", directory, blobKey)
}

func GetBlobName(tenantID string, docID string) string {
	return fmt.Sprintf("%s/%s", tenantID, docID)
}
