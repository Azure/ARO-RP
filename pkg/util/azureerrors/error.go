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
	CODE_AUTHFAILED       = "AuthorizationFailed"
	CODE_DEPLOYACTIVE     = "DeploymentActive"
	CODE_DEPLOYFAILED     = "DeploymentFailed"
	CODE_FORBIDDEN        = "Forbidden"
	CODE_INVALIDTEMPL     = "InvalidTemplateDeployment"
	CODE_LINKEDAUTHFAILED = "LinkedAuthorizationFailed"
	CODE_RGNOTFOUND       = "ResourceGroupNotFound"

	// VM SKU availability error codes
	CODE_INVALIDPARAM          = "InvalidParameter"
	CODE_NOTAVAILABLEFORSUBSCR = "NotAvailableForSubscription"
	CODE_QUOTAEXCEEDED         = "QuotaExceeded"
	CODE_SKUNOTAVAILABLE       = "SkuNotAvailable"
)

// VMProfileType identifies which VM profile (master or worker) is affected by an error.
type VMProfileType int

const (
	VMProfileUnknown VMProfileType = iota
	VMProfileMaster
	VMProfileWorker
)

// HasAuthorizationFailedError returns true it the error is, or contains, an
// AuthorizationFailed error
func HasAuthorizationFailedError(err error) bool {
	return deploymentFailedDueToAuthError(err, CODE_AUTHFAILED)
}

// HasLinkedAuthorizationFailedError returns true it the error is, or contains, a
// LinkedAuthorizationFailed error
func HasLinkedAuthorizationFailedError(err error) bool {
	return deploymentFailedDueToAuthError(err, CODE_LINKEDAUTHFAILED)
}

func deploymentFailedDueToAuthError(err error, authCode string) bool {
	// Check go-autorest SDK errors (old SDK)
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
		serviceErr.Code == CODE_DEPLOYFAILED {
		for _, d := range serviceErr.Details {
			if code, ok := d["code"].(string); ok &&
				code == CODE_FORBIDDEN {
				if message, ok := d["message"].(string); ok {
					var ce *api.CloudError
					if json.Unmarshal([]byte(message), &ce) == nil &&
						ce.CloudErrorBody != nil &&
						ce.Code == authCode {
						return true
					}
				}
			}
		}
	}

	// Check azcore SDK errors (new SDK)
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		if responseError.ErrorCode == authCode {
			return true
		}
	}

	return false
}

// IsDeploymentMissingPermissionsError returns true if the error indicates that
// ARM rejected a template deployment pre-flight due to missing role
// assignments. This can be an indicator of role assignment propagation delay.
func IsDeploymentMissingPermissionsError(err error) bool {
	// Check go-autorest SDK errors (old SDK)
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			if serviceErr.Code == CODE_INVALIDTEMPL && strings.Contains(serviceErr.Message, "Authorization failed for template resource") {
				return true
			}
		}
	}

	// Check azcore SDK errors (new SDK)
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		if responseError.ErrorCode == CODE_INVALIDTEMPL && strings.Contains(err.Error(), "Authorization failed for template resource") {
			return true
		}
	}

	return false
}

// IsDeploymentActiveError returns true it the error is a DeploymentActive
// error.
func IsDeploymentActiveError(err error) bool {
	// Check go-autorest SDK errors (old SDK)
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == CODE_DEPLOYACTIVE {
			return true
		}
	}

	// Check azcore SDK errors (new SDK)
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		if responseError.ErrorCode == CODE_DEPLOYACTIVE {
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
	// Check go-autorest SDK errors (old SDK)
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			if serviceErr.Code == CODE_RGNOTFOUND {
				return true
			}
		}
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == CODE_RGNOTFOUND {
			return true
		}
	}

	// Check azcore SDK errors (new SDK)
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		if responseError.ErrorCode == CODE_RGNOTFOUND {
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

// IsVMSKUError checks if the error is a VM SKU availability error and returns
// which profile (master/worker) is affected.
// Azure Resource Manager error codes: https://learn.microsoft.com/en-us/azure/azure-resource-manager/troubleshooting/error-sku-not-available
func IsVMSKUError(err error) (bool, VMProfileType) {
	if err == nil {
		return false, VMProfileUnknown
	}

	// Check azcore SDK errors (new SDK)
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		if responseError.ErrorCode == CODE_SKUNOTAVAILABLE ||
			responseError.ErrorCode == CODE_NOTAVAILABLEFORSUBSCR ||
			responseError.ErrorCode == CODE_QUOTAEXCEEDED {
			return true, detectVMProfile(err.Error())
		}
		// ARO RP validation error
		if responseError.ErrorCode == CODE_INVALIDPARAM && strings.Contains(err.Error(), "SKU") {
			return true, detectVMProfile(err.Error())
		}
	}

	// Check go-autorest SDK errors (old SDK)
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			if serviceErr.Code == CODE_SKUNOTAVAILABLE ||
				serviceErr.Code == CODE_NOTAVAILABLEFORSUBSCR ||
				serviceErr.Code == CODE_QUOTAEXCEEDED {
				return true, detectVMProfile(err.Error())
			}
			// ARO RP validation error
			if serviceErr.Code == CODE_INVALIDPARAM && strings.Contains(serviceErr.Message, "SKU") {
				return true, detectVMProfile(err.Error())
			}
		}
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil {
			if requestErr.ServiceError.Code == CODE_SKUNOTAVAILABLE ||
				requestErr.ServiceError.Code == CODE_NOTAVAILABLEFORSUBSCR ||
				requestErr.ServiceError.Code == CODE_QUOTAEXCEEDED {
				return true, detectVMProfile(err.Error())
			}
			if requestErr.ServiceError.Code == CODE_INVALIDPARAM && strings.Contains(requestErr.ServiceError.Message, "SKU") {
				return true, detectVMProfile(err.Error())
			}
		}
	}

	errStr := err.Error()
	if strings.Contains(errStr, CODE_SKUNOTAVAILABLE) ||
		strings.Contains(errStr, CODE_NOTAVAILABLEFORSUBSCR) ||
		strings.Contains(errStr, CODE_QUOTAEXCEEDED) {
		return true, detectVMProfile(errStr)
	}
	if strings.Contains(errStr, CODE_INVALIDPARAM) && strings.Contains(errStr, "SKU") {
		return true, detectVMProfile(errStr)
	}
	if strings.Contains(errStr, "not available in location") && strings.Contains(errStr, "size") {
		return true, detectVMProfile(errStr)
	}

	return false, VMProfileUnknown
}

func detectVMProfile(errStr string) VMProfileType {
	if strings.Contains(errStr, "workerProfiles") || strings.Contains(errStr, "WorkerProfiles") {
		return VMProfileWorker
	}
	if strings.Contains(errStr, "masterProfile") || strings.Contains(errStr, "MasterProfile") {
		return VMProfileMaster
	}
	return VMProfileUnknown
}

// IsRetryableError returns true if the error is a transient/retryable error
// such as 429 Too Many Requests or contains RetryableError code
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		if responseError.StatusCode == http.StatusTooManyRequests {
			return true
		}
	}

	var detailedErr autorest.DetailedError
	if errors.As(err, &detailedErr) {
		if detailedErr.StatusCode == http.StatusTooManyRequests {
			return true
		}
	}

	// Check for RetryableError in error message (nested Azure errors)
	return strings.Contains(err.Error(), "RetryableError")
}
