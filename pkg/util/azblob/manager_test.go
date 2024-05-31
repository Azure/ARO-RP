package azblob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	azstorage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/golang/mock/gomock"

	mock_armstorage "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armstorage"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type fakeTokenCredential struct{}

func (c fakeTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{}, nil
}

type fakeReadCloser struct {
	io.Reader
}

func (fakeReadCloser) Close() error { return nil }

func TestCreateBlobContainer(t *testing.T) {
	ctx := context.Background()
	resourceGroupName := "fakeResourceGroup"
	containerName := "fakeContainer"

	container := azstorage.BlobContainer{
		ContainerProperties: &azstorage.ContainerProperties{
			PublicAccess: to.Ptr(azstorage.PublicAccessNone),
		},
	}
	respErrContainerNotFound := &azcore.ResponseError{
		ErrorCode: string(bloberror.ContainerNotFound),
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
		mocks   func(*mock_armstorage.MockBlobContainersClient)
		wantErr string
	}{
		{
			name: "Success - Create the container",
			mocks: func(blobContainer *mock_armstorage.MockBlobContainersClient) {
				blobContainer.EXPECT().Get(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientGetOptions{}).Return(azstorage.BlobContainersClientGetResponse{}, respErrContainerNotFound)
				blobContainer.EXPECT().Create(ctx, resourceGroupName, "", containerName, container, &azstorage.BlobContainersClientCreateOptions{}).Return(azstorage.BlobContainersClientCreateResponse{}, nil)
			},
		},
		{
			name: "Success - Container already exists, so not creating",
			mocks: func(blobContainer *mock_armstorage.MockBlobContainersClient) {
				blobContainer.EXPECT().Get(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientGetOptions{}).Return(azstorage.BlobContainersClientGetResponse{}, nil)
			},
		},
		{
			name: "Fail - Get Container fails with generic error",
			mocks: func(blobContainer *mock_armstorage.MockBlobContainersClient) {
				blobContainer.EXPECT().Get(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientGetOptions{}).Return(azstorage.BlobContainersClientGetResponse{}, respErrGeneric)
			},
			wantErr: genericErrorMessage,
		},
		{
			name: "Fail - Create Container fails with generic error",
			mocks: func(blobContainer *mock_armstorage.MockBlobContainersClient) {
				blobContainer.EXPECT().Get(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientGetOptions{}).Return(azstorage.BlobContainersClientGetResponse{}, respErrContainerNotFound)
				blobContainer.EXPECT().Create(ctx, resourceGroupName, "", containerName, container, &azstorage.BlobContainersClientCreateOptions{}).Return(azstorage.BlobContainersClientCreateResponse{}, respErrGeneric)
			},
			wantErr: genericErrorMessage,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			blobContainer := mock_armstorage.NewMockBlobContainersClient(controller)

			if tt.mocks != nil {
				tt.mocks(blobContainer)
			}

			m := &manager{
				cred:          fakeTokenCredential{},
				blobContainer: blobContainer,
			}

			err := m.CreateBlobContainer(ctx, resourceGroupName, "", containerName, azstorage.PublicAccessNone)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeleteBlobContainer(t *testing.T) {
	ctx := context.Background()
	resourceGroupName := "fakeResourceGroup"
	containerName := "fakeContainer"
	respErrContainerNotFound := &azcore.ResponseError{
		ErrorCode: string(bloberror.ContainerNotFound),
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
		mocks   func(*mock_armstorage.MockBlobContainersClient)
		wantErr string
	}{
		{
			name: "Success - Delete the container",
			mocks: func(blobContainer *mock_armstorage.MockBlobContainersClient) {
				blobContainer.EXPECT().Get(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientGetOptions{}).Return(azstorage.BlobContainersClientGetResponse{}, nil)
				blobContainer.EXPECT().Delete(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientDeleteOptions{}).Return(azstorage.BlobContainersClientDeleteResponse{}, nil)
			},
		},
		{
			name: "Success - Container does not exist, so not deleting",
			mocks: func(blobContainer *mock_armstorage.MockBlobContainersClient) {
				blobContainer.EXPECT().Get(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientGetOptions{}).Return(azstorage.BlobContainersClientGetResponse{}, respErrContainerNotFound)
			},
		},
		{
			name: "Success - Get Container fails with generic error, still attempt container deletion",
			mocks: func(blobContainer *mock_armstorage.MockBlobContainersClient) {
				blobContainer.EXPECT().Get(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientGetOptions{}).Return(azstorage.BlobContainersClientGetResponse{}, respErrGeneric)
				blobContainer.EXPECT().Delete(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientDeleteOptions{}).Return(azstorage.BlobContainersClientDeleteResponse{}, nil)
			},
		},
		{
			name: "Fail - Delete Container fails with generic error",
			mocks: func(blobContainer *mock_armstorage.MockBlobContainersClient) {
				blobContainer.EXPECT().Get(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientGetOptions{}).Return(azstorage.BlobContainersClientGetResponse{}, nil)
				blobContainer.EXPECT().Delete(ctx, resourceGroupName, "", containerName, &azstorage.BlobContainersClientDeleteOptions{}).Return(azstorage.BlobContainersClientDeleteResponse{}, respErrGeneric)
			},
			wantErr: genericErrorMessage,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			blobContainer := mock_armstorage.NewMockBlobContainersClient(controller)

			if tt.mocks != nil {
				tt.mocks(blobContainer)
			}

			m := &manager{
				cred:          fakeTokenCredential{},
				blobContainer: blobContainer,
			}

			err := m.DeleteBlobContainer(ctx, resourceGroupName, "", containerName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
