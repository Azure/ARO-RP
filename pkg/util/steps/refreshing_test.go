package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type expectCloudErrorFields struct {
	statusCode int
	code       string
	target     string
	message    string
}

func TestCreateActionableError(t *testing.T) {
	for _, tt := range []struct {
		testName         string
		rawErr           error
		managedRGName    string
		expectCloudError *expectCloudErrorFields
	}{
		{
			testName: "Should not return a CloudError when original error is nil",
			rawErr:   nil,
		},
		{
			testName: "Should return the error if it is not convertible to user actionable one",
			rawErr:   errors.New("unknown or unhandled error"),
		},
		{
			testName: "Should return a CloudError when original error is AADSTS700016",
			rawErr:   errors.New("AADSTS700016"),
			expectCloudError: &expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"The provided service principal application (client) ID was not found in the directory (tenant). Please ensure that the provided application (client) id and client secret value are correct.",
			},
		},
		{
			testName: "Should return a CloudError when original error is AuthorizationFailed",
			rawErr: &azure.ServiceError{
				Code:    "DeploymentFailed",
				Message: "Unknown service error",
				Details: []map[string]interface{}{
					{
						"code":    "Forbidden",
						"message": "{\"error\": {\"code\": \"AuthorizationFailed\"} }",
					},
				},
			},
			expectCloudError: &expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"Authorization using provided credentials failed. Please ensure that the provided application (client) id and client secret value are correct.",
			},
		},
		{
			testName: "Should return a CloudError when original error is AADSTS7000222",
			rawErr:   errors.New("AADSTS7000222"),
			expectCloudError: &expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"The provided client secret is expired. Please create a new one for your service principal.",
			},
		},
		{
			testName: "Should return InvalidSecretError as SP credentials error",
			rawErr:   errors.New("AADSTS7000215"),
			expectCloudError: &expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"Authorization using provided credentials failed. Please ensure that the provided application (client) id and client secret value are correct.",
			},
		},
		{
			testName:      "AuthorizationFailed on managed RG returns InternalServerError",
			managedRGName: "aro-managed-rg",
			rawErr: autorest.DetailedError{
				Original: &azure.ServiceError{
					Code:    "AuthorizationFailed",
					Message: "The client does not have authorization to perform action over scope '/subscriptions/sub/resourceGroups/aro-managed-rg/providers/Microsoft.Network/loadBalancers/lb'",
				},
				StatusCode: http.StatusForbidden,
				Response: &http.Response{
					Request: &http.Request{
						URL: &url.URL{
							Path: "/subscriptions/sub/resourceGroups/aro-managed-rg/providers/Microsoft.Network/loadBalancers/lb",
						},
					},
				},
			},
			expectCloudError: &expectCloudErrorFields{
				http.StatusInternalServerError,
				api.CloudErrorCodeInternalServerError,
				"",
				"Internal server error.",
			},
		},
		{
			testName:      "AuthorizationFailed on customer RG still returns InvalidServicePrincipalCredentials",
			managedRGName: "aro-managed-rg",
			rawErr: autorest.DetailedError{
				Original: &azure.ServiceError{
					Code:    "AuthorizationFailed",
					Message: "The client does not have authorization to perform action over scope '/subscriptions/sub/resourceGroups/customer-vnet-rg/providers/Microsoft.Network/virtualNetworks/vnet'",
				},
				StatusCode: http.StatusForbidden,
				Response: &http.Response{
					Request: &http.Request{
						URL: &url.URL{
							Path: "/subscriptions/sub/resourceGroups/customer-vnet-rg/providers/Microsoft.Network/virtualNetworks/vnet",
						},
					},
				},
			},
			expectCloudError: &expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"Authorization using provided credentials failed. Please ensure that the provided application (client) id and client secret value are correct.",
			},
		},
		{
			testName:      "AuthorizationFailed with no managedRGName falls back to SP credentials error",
			managedRGName: "",
			rawErr: autorest.DetailedError{
				Original: &azure.ServiceError{
					Code:    "AuthorizationFailed",
					Message: "The client does not have authorization to perform action over scope '/subscriptions/sub/resourceGroups/aro-managed-rg/providers/Microsoft.Network/loadBalancers/lb'",
				},
				StatusCode: http.StatusForbidden,
				Response: &http.Response{
					Request: &http.Request{
						URL: &url.URL{
							Path: "/subscriptions/sub/resourceGroups/aro-managed-rg/providers/Microsoft.Network/loadBalancers/lb",
						},
					},
				},
			},
			expectCloudError: &expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"Authorization using provided credentials failed. Please ensure that the provided application (client) id and client secret value are correct.",
			},
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			err := CreateActionableError(tt.rawErr, tt.managedRGName)
			var cloudErr *api.CloudError
			if tt.expectCloudError != nil {
				isCloudErr := errors.As(err, &cloudErr)
				assert.True(t, isCloudErr)
				if isCloudErr {
					assert.Equal(t, tt.expectCloudError.statusCode, cloudErr.StatusCode)
					assert.Equal(t, tt.expectCloudError.code, cloudErr.Code)
					assert.Equal(t, tt.expectCloudError.target, cloudErr.Target)
					assert.Equal(t, tt.expectCloudError.message, cloudErr.Message)
				}
			} else {
				assert.Equal(t, err, tt.rawErr)
			}
		})
	}
}

type fakeRefreshableAuthorizer struct {
	rebuildCalled int
	rebuildErr    error
}

func (f *fakeRefreshableAuthorizer) Rebuild() error {
	f.rebuildCalled++
	return f.rebuildErr
}

func (f *fakeRefreshableAuthorizer) WithAuthorization() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer { return p }
}

func TestAuthorizationRefreshingActionRetries(t *testing.T) {
	forbiddenErr := &azcore.ResponseError{StatusCode: http.StatusForbidden}
	rebuildErr := errors.New("authorizer rebuild failed")

	for _, tt := range []struct {
		name           string
		errors         []error
		auth           *fakeRefreshableAuthorizer
		retryTimeout   time.Duration
		pollInterval   time.Duration
		repeatLastErr  bool
		expectRetries  bool
		expectRebuild  bool
		expectFinalErr string
		expectError    error
	}{
		{
			name:           "AuthorizationFailed is retried then succeeds",
			errors:         []error{autorest.DetailedError{Original: &azure.ServiceError{Code: "AuthorizationFailed"}}, nil},
			auth:           &fakeRefreshableAuthorizer{},
			expectRetries:  true,
			expectFinalErr: "",
		},
		{
			name:           "LinkedAuthorizationFailed is retried then succeeds",
			errors:         []error{autorest.DetailedError{Original: &azure.ServiceError{Code: "LinkedAuthorizationFailed"}}, nil},
			auth:           &fakeRefreshableAuthorizer{},
			expectRetries:  true,
			expectFinalErr: "",
		},
		{
			name:           "UnauthorizedClient (AADSTS700016) is retried then succeeds",
			errors:         []error{fmt.Errorf("AADSTS700016: application not found"), nil},
			auth:           &fakeRefreshableAuthorizer{},
			expectRetries:  true,
			expectFinalErr: "",
		},
		{
			name:           "InvalidSecret (AADSTS7000215) is retried then succeeds",
			errors:         []error{fmt.Errorf("AADSTS7000215: invalid client secret"), nil},
			auth:           &fakeRefreshableAuthorizer{},
			expectRetries:  true,
			expectFinalErr: "",
		},
		{
			name: "DeploymentMissingPermissions is retried then succeeds",
			errors: []error{
				autorest.DetailedError{Original: &azure.ServiceError{Code: "InvalidTemplateDeployment", Message: "Authorization failed for template resource 'foo'"}},
				nil,
			},
			auth:           &fakeRefreshableAuthorizer{},
			expectRetries:  true,
			expectFinalErr: "",
		},
		{
			name:           "ErrWantRefresh is retried then succeeds",
			errors:         []error{ErrWantRefresh, nil},
			auth:           &fakeRefreshableAuthorizer{},
			expectRetries:  true,
			expectFinalErr: "",
		},
		{
			name:           "non-auth error is not retried",
			errors:         []error{fmt.Errorf("some other error")},
			auth:           &fakeRefreshableAuthorizer{},
			expectRetries:  false,
			expectFinalErr: "some other error",
		},
		{
			name:           "nil error succeeds immediately",
			errors:         []error{nil},
			auth:           &fakeRefreshableAuthorizer{},
			expectRetries:  false,
			expectFinalErr: "",
		},
		{
			name:          "forbidden error without authorizer is retried then succeeds",
			errors:        []error{forbiddenErr, nil},
			expectRetries: true,
		},
		{
			name:         "forbidden error without authorizer returns last error after retry timeout",
			errors:       []error{forbiddenErr},
			retryTimeout: time.Millisecond,
			pollInterval: 30 * time.Second,
			expectError:  forbiddenErr,
		},
		{
			name:          "persistent forbidden error with authorizer returns error after retry timeout",
			errors:        []error{forbiddenErr},
			auth:          &fakeRefreshableAuthorizer{},
			retryTimeout:  100 * time.Millisecond,
			pollInterval:  30 * time.Second,
			repeatLastErr: true,
			expectRebuild: true,
			expectError:   forbiddenErr,
		},
		{
			name:          "authorizer rebuild error is returned",
			errors:        []error{forbiddenErr},
			auth:          &fakeRefreshableAuthorizer{rebuildErr: rebuildErr},
			expectRebuild: true,
			expectError:   rebuildErr,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.LogForTesting(t)
			retryTimeout := tt.retryTimeout
			if retryTimeout == 0 {
				retryTimeout = time.Second
			}
			pollInterval := tt.pollInterval
			if pollInterval == 0 {
				pollInterval = time.Millisecond
			}

			callCount := 0
			action := func(ctx context.Context) error {
				idx := callCount
				callCount++
				if idx < len(tt.errors) {
					return tt.errors[idx]
				}
				if tt.repeatLastErr {
					return tt.errors[len(tt.errors)-1]
				}
				return nil
			}

			s := &authorizationRefreshingActionStep{
				f:             action,
				retryTimeout:  retryTimeout,
				pollInterval:  pollInterval,
				managedRGName: "",
			}
			if tt.auth != nil {
				s.auth = tt.auth
			}

			err := s.run(t.Context(), log)

			if tt.expectRetries {
				assert.Greater(t, callCount, 1, "action should have been called more than once")
			} else {
				assert.Equal(t, 1, callCount, "action should have been called exactly once")
			}
			if tt.auth != nil {
				if tt.expectRetries || tt.expectRebuild {
					assert.Positive(t, tt.auth.rebuildCalled, "Rebuild should have been called")
				} else {
					assert.Equal(t, 0, tt.auth.rebuildCalled, "Rebuild should not have been called")
				}
			}

			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)
			} else if tt.expectFinalErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.expectFinalErr)
			}
		})
	}
}
