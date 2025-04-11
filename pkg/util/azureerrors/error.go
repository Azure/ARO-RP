package azureerrors

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	CodeInvalidTemplateDeployment = "InvalidTemplateDeployment"
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

// IsDeploymentMissingPermissionsError returns true if the error indicates that
// ARM rejected a template deployment pre-flight due to missing role
// assignments.
// This can be an indicator of role assignment propagation delay.
func IsDeploymentMissingPermissionsError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			if serviceErr.Code == CodeInvalidTemplateDeployment && strings.Contains(serviceErr.Message, "Authorization failed for template resource") {
				return true
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

func IsNotFoundError(err error) bool {
	var detailedErr autorest.DetailedError
	if errors.As(err, &detailedErr) {
		return detailedErr.StatusCode == http.StatusNotFound
	}

	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		return responseError.StatusCode == http.StatusNotFound
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

// Returns true when the error is due to expired application client/secret keys.
// See https://learn.microsoft.com/en-us/entra/identity-platform/reference-error-codes#aadsts-error-codes
func IsClientSecretKeysExpired(err error) bool {
	return strings.Contains(err.Error(), "AADSTS7000222")
}

// ResourceGroupNotFound returns true if the error is an ResourceGroupNotFound error
func ResourceGroupNotFound(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			if serviceErr.Code == "ResourceGroupNotFound" {
				return true
			}
		}
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "ResourceGroupNotFound" {
			return true
		}
	}
	return false
}

func Is4xxError(err error) bool {
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		return responseError.StatusCode >= 400 && responseError.StatusCode < 500
	}
	return false
}
