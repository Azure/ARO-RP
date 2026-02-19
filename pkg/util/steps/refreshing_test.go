package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
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
			testName:      "AuthorizationFailed on managed RG returns InvalidResourceProviderPermissions",
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
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidResourceProviderPermissions,
				"",
				"The resource provider does not have enough permissions on the managed resource group. Please check that the resource provider permissions are correct.",
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
