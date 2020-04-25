package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
)

// hasAuthorizationFailedError returns true it the error is, or contains, an
// AuthorizationFailed error
func hasAuthorizationFailedError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			if serviceErr.Code == "AuthorizationFailed" {
				return true
			}
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
						ce.CloudErrorBody.Code == "AuthorizationFailed" {
						return true
					}
				}
			}
		}
	}

	return false
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
