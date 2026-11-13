package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	storagesdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armstorage "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armstorage"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestBlobServiceReturnsErrorsWithoutApplicationRetry(t *testing.T) {
	for _, tt := range []struct {
		name       string
		statusCode int
	}{
		{name: "throttling", statusCode: http.StatusTooManyRequests},
		{name: "permanent error", statusCode: http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			responseError := &azcore.ResponseError{
				StatusCode: tt.statusCode,
				ErrorCode:  http.StatusText(tt.statusCode),
			}
			accounts := mock_armstorage.NewMockAccountsClient(gomock.NewController(t))
			accounts.EXPECT().
				ListAccountSAS(ctx, "rg", "account", gomock.Any(), nil).
				Return(storagesdk.AccountsClientListAccountSASResponse{}, responseError).Times(1)
			m := &manager{storageAccounts: accounts, storageEndpointSuffix: "core.windows.net", clientOptions: &arm.ClientOptions{}}
			client, err := m.BlobService(ctx, "rg", "account", storagesdk.PermissionsR, storagesdk.SignedResourceTypesO)
			if client != nil {
				t.Fatal("Expected no client on failure")
			}
			switch tt.statusCode {
			case http.StatusTooManyRequests:
				var cloudError *api.CloudError
				if !errors.As(err, &cloudError) || cloudError.Code != api.CloudErrorCodeThrottlingLimitExceeded ||
					len(cloudError.Details) != 1 || !strings.Contains(cloudError.Details[0].Message, "Azure Storage limits") {
					t.Fatalf("Expected storage throttling error, got %v", err)
				}
			default:
				if err != responseError {
					t.Fatalf("Expected original permanent error, got %v", err)
				}
			}
		})
	}
}

func TestGetCorrectErrWhenTooManyRequests(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
	}{
		{
			name: "No Error",
		},
		{
			name: "Too Many Requests Error",
			err: &azcore.ResponseError{
				StatusCode: http.StatusTooManyRequests,
				ErrorCode:  "TooManyRequests",
			},
		},
		{
			name: "Wrapped Too Many Requests Error",
			err: fmt.Errorf("listing SAS: %w", &azcore.ResponseError{
				StatusCode: http.StatusTooManyRequests,
				ErrorCode:  "TooManyRequests",
			}),
		},
		{
			name: "Non-throttling Azure Error",
			err: &azcore.ResponseError{
				StatusCode: http.StatusForbidden,
				ErrorCode:  "AuthorizationFailed",
			},
		},
		{
			name: "Not a azcore.ResponseError",
			err:  errors.New("Test Error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resErr := getCorrectErrWhenTooManyRequests(tt.err)
			responseError, isResponseError := tt.err.(*azcore.ResponseError)
			if isResponseError && responseError.StatusCode == http.StatusTooManyRequests {
				cloudError, ok := resErr.(*api.CloudError)
				if !ok {
					t.Fatalf("Expected public CloudError, got %T", resErr)
				}
				if cloudError.StatusCode != http.StatusTooManyRequests || cloudError.Code != api.CloudErrorCodeThrottlingLimitExceeded {
					t.Fatalf("Unexpected throttling error: %v", cloudError)
				}
				if len(cloudError.Details) != 1 || !strings.Contains(cloudError.Details[0].Message, "Azure Storage limits") {
					t.Fatalf("Expected original storage limits message, got %v", cloudError.Details)
				}
				return
			}
			if resErr != tt.err {
				t.Fatalf("Expected original error %v, got %v", tt.err, resErr)
			}
		})
	}
}
