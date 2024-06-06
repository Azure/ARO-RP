package oidcbuilder

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	utilazblob "github.com/Azure/ARO-RP/pkg/util/azblob"
)

const (
	BodyKey = ".well-known/openid-configuration"
	JWKSKey = "openid/v1/jwks"
)

type OIDCBuilder struct {
	privateKey       []byte
	publicKey        []byte
	blobContainerURL string
	endpointURL      string
}

func NewOIDCBuilder(storageEndpointSuffix string, storageEndpoint string, accountName, containerName string) (*OIDCBuilder, error) {
	privateKey, publicKey, err := CreateKeyPair()
	if err != nil {
		return nil, err
	}

	return &OIDCBuilder{
		privateKey:       privateKey,
		publicKey:        publicKey,
		blobContainerURL: fmt.Sprintf("https://%s.blob.%s/%s", accountName, storageEndpointSuffix, containerName),
		endpointURL:      fmt.Sprintf("https://%s/%s", storageEndpoint, containerName),
	}, nil
}

func (b *OIDCBuilder) EnsureOIDCDocs(ctx context.Context, oidcContainerName string, azBlobClient utilazblob.AZBlobClient) error {
	// Create the OIDC configuration
	discoveryDocument := GenerateDiscoveryDocument(b.endpointURL)

	// Create the OIDC key list
	jwks, err := BuildJSONWebKeySet(b.publicKey)
	if err != nil {
		return err
	}

	return populateOidcFolder(ctx, discoveryDocument, jwks, azBlobClient)
}

func (b *OIDCBuilder) GetEndpointUrl() string {
	return b.endpointURL
}

func (b *OIDCBuilder) GetPrivateKey() []byte {
	return b.privateKey
}

func (b *OIDCBuilder) GetBlobContainerURL() string {
	return b.blobContainerURL
}

func populateOidcFolder(ctx context.Context, body string, jwks []byte, azBlobClient utilazblob.AZBlobClient) error {
	err := azBlobClient.UploadBuffer(
		ctx,
		"",
		BodyKey,
		[]byte(body),
	)
	if err != nil {
		return err
	}

	return azBlobClient.UploadBuffer(
		ctx,
		"",
		JWKSKey,
		jwks,
	)
}
