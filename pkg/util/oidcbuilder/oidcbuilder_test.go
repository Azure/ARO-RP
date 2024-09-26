package oidcbuilder

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_azblob "github.com/Azure/ARO-RP/pkg/util/mocks/azblob"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestEnsureOIDCDocs(t *testing.T) {
	ctx := context.Background()
	directoryName := "fakeDirectory"
	blobContainerURL := "fakeBlobContainerURL"
	endpointURL := "fakeEndPointURL"

	priKey, pubKey, incorrectlyEncodedPublicKey := getTestKeyData(t)

	nonRSAPrivateKey, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	nonRSAPubKeyBytes, err := x509.MarshalPKIXPublicKey(&nonRSAPrivateKey.PublicKey)
	nonRSAEncodedPublicKey := pem.EncodeToMemory(&pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   nonRSAPubKeyBytes,
	})

	invalidKey := []byte("Invalid Key")

	for _, tt := range []struct {
		name        string
		mocks       func(*mock_azblob.MockAZBlobClient)
		oidcbuilder *OIDCBuilder
		wantErr     string
	}{
		{
			name: "Success",
			oidcbuilder: &OIDCBuilder{
				privateKey:       priKey,
				publicKey:        pubKey,
				blobContainerURL: blobContainerURL,
				directory:        directoryName,
				endpointURL:      endpointURL,
			},
			mocks: func(azblobClient *mock_azblob.MockAZBlobClient) {
				azblobClient.EXPECT().
					UploadBuffer(gomock.Any(), "", DocumentKey(directoryName, DiscoveryDocumentKey), gomock.Any()).
					Return(nil)
				azblobClient.EXPECT().
					UploadBuffer(gomock.Any(), "", DocumentKey(directoryName, JWKSKey), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "Fail -Invalid Public Key fails during decoding",
			oidcbuilder: &OIDCBuilder{
				privateKey:       priKey,
				publicKey:        invalidKey,
				blobContainerURL: blobContainerURL,
				endpointURL:      endpointURL,
				directory:        directoryName,
			},
			wantErr: "Failed to decode PEM file",
		},
		{
			name: "Fail - Valid Public Key(PEM) but not expected type",
			oidcbuilder: &OIDCBuilder{
				privateKey:       priKey,
				publicKey:        incorrectlyEncodedPublicKey,
				blobContainerURL: blobContainerURL,
				endpointURL:      endpointURL,
				directory:        directoryName,
			},
			wantErr: "Failed to parse key content: x509: failed to parse public key (use ParsePKCS1PublicKey instead for this key format)",
		},
		{
			name: "Fail - Error when uploading OIDC main configuration",
			oidcbuilder: &OIDCBuilder{
				privateKey:       priKey,
				publicKey:        pubKey,
				blobContainerURL: blobContainerURL,
				endpointURL:      endpointURL,
				directory:        directoryName,
			},
			mocks: func(azblobClient *mock_azblob.MockAZBlobClient) {
				azblobClient.EXPECT().
					UploadBuffer(gomock.Any(), "", DocumentKey(directoryName, DiscoveryDocumentKey), gomock.Any()).
					Return(errors.New("generic error"))
			},
			wantErr: "generic error",
		},
		{
			name: "Fail - Error when uploading JWKS",
			oidcbuilder: &OIDCBuilder{
				privateKey:       priKey,
				publicKey:        pubKey,
				blobContainerURL: blobContainerURL,
				endpointURL:      endpointURL,
				directory:        directoryName,
			},
			mocks: func(azblobClient *mock_azblob.MockAZBlobClient) {
				azblobClient.EXPECT().
					UploadBuffer(gomock.Any(), "", DocumentKey(directoryName, DiscoveryDocumentKey), gomock.Any()).
					Return(nil)
				azblobClient.EXPECT().
					UploadBuffer(gomock.Any(), "", DocumentKey(directoryName, JWKSKey), gomock.Any()).
					Return(errors.New("generic error"))
			},
			wantErr: "generic error",
		},
		{
			name: "Fail - Public key is not of type RSA",
			oidcbuilder: &OIDCBuilder{
				privateKey:       priKey,
				publicKey:        nonRSAEncodedPublicKey,
				blobContainerURL: blobContainerURL,
				endpointURL:      endpointURL,
				directory:        directoryName,
			},
			wantErr: "Public key is not of type RSA",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			azBlobClient := mock_azblob.NewMockAZBlobClient(controller)

			if tt.mocks != nil {
				tt.mocks(azBlobClient)
			}

			err = tt.oidcbuilder.EnsureOIDCDocs(ctx, azBlobClient)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.oidcbuilder.GetEndpointUrl() != tt.oidcbuilder.endpointURL {
				t.Fatalf("GetEndpointUrl doesn't match the original endpointURL - %s != %s (wanted)", tt.oidcbuilder.GetEndpointUrl(), tt.oidcbuilder.endpointURL)
			}

			if !reflect.DeepEqual(string(tt.oidcbuilder.privateKey), tt.oidcbuilder.GetPrivateKey()) {
				t.Fatalf("GetPrivateKey doesn't match the original privateKey")
			}

			if tt.oidcbuilder.GetBlobContainerURL() != tt.oidcbuilder.blobContainerURL {
				t.Fatalf("GetBlobContainerURL doesn't match the original endpointURL - %s != %s (wanted)", tt.oidcbuilder.GetBlobContainerURL(), tt.oidcbuilder.blobContainerURL)
			}
		})
	}
}

func getTestKeyData(t *testing.T) ([]byte, []byte, []byte) {
	t.Helper()

	testKeyBitSize := 256

	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().OIDCKeyBitSize().Return(testKeyBitSize)
	priKey, pubKey, err := CreateKeyPair(env)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, testKeyBitSize)
	if err != nil {
		t.Fatal(err)
	}
	pubKeyBytes := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)
	incorrectlyEncodedPublicKey := pem.EncodeToMemory(&pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   pubKeyBytes,
	})
	return priKey, pubKey, incorrectlyEncodedPublicKey
}

type fakeReadCloser struct {
	io.Reader
}

func (fakeReadCloser) Close() error { return nil }
func TestDeleteOidcFolder(t *testing.T) {
	ctx := context.Background()
	directoryName := "fakeDirectory"
	respErrBlobNotFound := &azcore.ResponseError{
		ErrorCode: string(bloberror.BlobNotFound),
	}
	respErrGeneric := &azcore.ResponseError{
		ErrorCode: string("Generic Error"),
		RawResponse: &http.Response{
			Request: &http.Request{
				Method: "FAKE",
				URL:    &url.URL{},
			},
			Body: fakeReadCloser{bytes.NewBufferString("Generic Error")},
		},
		StatusCode: 400,
	}
	genericErrorMessage := `FAKE ://
--------------------------------------------------------------------------------
RESPONSE 0: 
ERROR CODE: Generic Error
--------------------------------------------------------------------------------
Generic Error
--------------------------------------------------------------------------------
`

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_azblob.MockAZBlobClient)
		wantErr string
	}{
		{
			name: "Success",
			mocks: func(azblobClient *mock_azblob.MockAZBlobClient) {
				azblobClient.EXPECT().DeleteBlob(ctx, "", DocumentKey(directoryName, DiscoveryDocumentKey)).Return(nil)
				azblobClient.EXPECT().DeleteBlob(ctx, "", DocumentKey(directoryName, JWKSKey)).Return(nil)
			},
		},
		{
			name: "Fail - Generic Error when deleting DiscoveryDocument",
			mocks: func(azblobClient *mock_azblob.MockAZBlobClient) {
				azblobClient.EXPECT().DeleteBlob(ctx, "", DocumentKey(directoryName, DiscoveryDocumentKey)).Return(respErrGeneric)
			},
			wantErr: genericErrorMessage,
		},
		{
			name: "Fail - Generic Error when deleting JWKS",
			mocks: func(azblobClient *mock_azblob.MockAZBlobClient) {
				azblobClient.EXPECT().DeleteBlob(ctx, "", DocumentKey(directoryName, DiscoveryDocumentKey)).Return(respErrBlobNotFound)
				azblobClient.EXPECT().DeleteBlob(ctx, "", DocumentKey(directoryName, JWKSKey)).Return(respErrGeneric)
			},
			wantErr: genericErrorMessage,
		},
		{
			name: "Success - One Blob exists and other doesn't",
			mocks: func(azblobClient *mock_azblob.MockAZBlobClient) {
				azblobClient.EXPECT().DeleteBlob(ctx, "", DocumentKey(directoryName, DiscoveryDocumentKey)).Return(respErrBlobNotFound)
				azblobClient.EXPECT().DeleteBlob(ctx, "", DocumentKey(directoryName, JWKSKey)).Return(nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			azBlobClient := mock_azblob.NewMockAZBlobClient(controller)

			if tt.mocks != nil {
				tt.mocks(azBlobClient)
			}

			err := DeleteOidcFolder(ctx, directoryName, azBlobClient)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestGenerateBlobContainerURL(t *testing.T) {
	oidcStorageAccountName := "eastusoic"
	for _, tt := range []struct {
		name     string
		mocks    func(*mock_env.MockInterface)
		expected string
	}{
		{
			name: "Success: Working as Expected",
			mocks: func(menv *mock_env.MockInterface) {
				menv.EXPECT().OIDCStorageAccountName().Return(oidcStorageAccountName)
				menv.EXPECT().Environment().Return(&azureclient.PublicCloud)
			},
			expected: fmt.Sprintf("https://%s.blob.%s/%s", oidcStorageAccountName, azureclient.PublicCloud.StorageEndpointSuffix, WebContainer),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)

			if tt.mocks != nil {
				tt.mocks(env)
			}

			result := GenerateBlobContainerURL(env)
			if result != tt.expected {
				t.Fatalf("Expected %s, but received %s", tt.expected, result)
			}
		})
	}
}
