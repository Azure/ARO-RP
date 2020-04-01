package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// isAuthorizationFailedError returns true it the error is an
// AuthorizationFailed error
func isAuthorizationFailedError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok &&
			serviceErr.Code == "AuthorizationFailed" {
			return true
		}
	}
	return false
}

// isResourceQuotaExceededError returns true and the original error message if
// the error is a QuotaExceeded error
func isResourceQuotaExceededError(err error) (bool, string) {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		// error format:
		// (autorest.DetailedError).Original.(*azure.ServiceError).Details.([]map[string]interface{})
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			for _, d := range serviceErr.Details {
				if code, ok := d["code"].(string); ok && code == "QuotaExceeded" {
					if message, ok := d["message"].(string); ok {
						return true, message
					}
				}
			}
		}
	}
	return false, ""
}

// isDeploymentActiveError returns true it the error is a DeploymentActive error
func isDeploymentActiveError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "DeploymentActive" {
			return true
		}
	}
	return false
}
