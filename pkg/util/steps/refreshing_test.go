package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

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
		expectCloudError *expectCloudErrorFields
	}{
		{
			"Should not return a CloudError when original error is nil",
			nil,
			nil,
		},
		{
			"Should return the error if it is not convertible to user actionable one",
			errors.New("unknown or unhandled error"),
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
				"The provided client secret is expired. Please create a new one for your service principal.",
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
			} else {
				assert.Equal(t, err, tt.rawErr)
			}
		})
	}
}
