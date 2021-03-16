package azureerrors

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
)

// HasAuthorizationFailedError returns true it the error is, or contains, an
// AuthorizationFailed error
func HasAuthorizationFailedError(err error) bool {
	return deploymentFailedDueToAuthError(err, "AuthorizationFailed")
}

// HasLinkedAuthorizationFailedError returns true it the error is, or contains, a
// LinkedAuthorizationFailed error
func HasLinkedAuthorizationFailedError(err error) bool {
	return deploymentFailedDueToAuthError(err, "LinkedAuthorizationFailed")
}

func deploymentFailedDueToAuthError(err error, authCode string) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			if serviceErr.Code == authCode {
				return true
			}
		}
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == authCode {
			return true
		}
	}

	if serviceErr, ok := err.(*azure.ServiceError); ok &&
		serviceErr.Code == "DeploymentFailed" {
		for _, d := range serviceErr.Details {
			if code, ok := d["code"].(string); ok &&
				code == "Forbidden" {
				if message, ok := d["message"].(string); ok {
					var ce *api.CloudError
					if json.Unmarshal([]byte(message), &ce) == nil &&
						ce.CloudErrorBody != nil &&
						ce.CloudErrorBody.Code == authCode {
						return true
					}
				}
			}
		}
	}

	return false
}

// IsDeploymentActiveError returns true it the error is a DeploymentActive error
func IsDeploymentActiveError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "DeploymentActive" {
			return true
		}
	}
	return false
}

// IsInvalidSecretError returns if errors is InvalidCredentials error
// Example: (adal.tokenRefreshError) adal: Refresh request failed. Status Code = '401'.
// Response body: {"error":"invalid_client","error_description":"AADSTS7000215:
// Invalid client secret is provided.
func IsInvalidSecretError(err error) bool {
	return strings.Contains(err.Error(), "AADSTS7000215")
}

// IsUnauthorizedClientError return if errors is UnauthorizedClient
// Example: {"error": "unauthorized_client", "error_description": "AADSTS700016:
// Application with identifier 'xxx' was not found in the directory 'xxx'. This
// can happen if the application has not been installed by the administrator of
// the tenant or consented to by any user in the tenant. You may have sent your
// authentication request to the wrong tenant. ...", "error_codes": [700016]}`.
// This can be an indicator of AAD propagation delay.
func IsUnauthorizedClientError(err error) bool {
	return strings.Contains(err.Error(), "AADSTS700016")
}
