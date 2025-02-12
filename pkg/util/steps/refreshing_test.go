package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
)

type expectCloudErrorFields struct {
	statusCode int
	code       string
	target     string
	message    string
}

func TestToActionableError(t *testing.T) {
	for _, tt := range []struct {
		testName         string
		rawErr           error
		expectCloudError *expectCloudErrorFields
	}{
		{
			"Should not return a CloudError when original error is nil",
			nil,
			nil,
		},
		{
			"Should return a CloudError when original error is AADSTS700016",
			errors.New("AADSTS700016"),
			&expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"The provided service principal application (client) ID was not found in the directory (tenant). Please ensure that the provided application (client) id and client secret value are correct.",
			},
		},
		{
			"Should return a CloudError when original error is AuthorizationFailed",
			&azure.ServiceError{
				Code:    "DeploymentFailed",
				Message: "Unknown service error",
				Details: []map[string]interface{}{
					{
						"code":    "Forbidden",
						"message": "{\"error\": {\"code\": \"AuthorizationFailed\"} }",
					},
				},
			},
			&expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"Authorization using provided credentials failed. Please ensure that the provided application (client) id and client secret value are correct.",
			},
		},
		{
			"Should return a CloudError when original error is AADSTS7000222",
			errors.New("AADSTS7000222"),
			&expectCloudErrorFields{
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"The provided application client and secret keys are expired. Please create new keys for your application.",
			},
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			err := CreateActionableError(tt.rawErr)
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
			}
		})
	}
}
