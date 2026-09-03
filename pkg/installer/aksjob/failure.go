package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"regexp"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/hive/failure"
)

var genericErr = &api.CloudError{
	StatusCode: http.StatusInternalServerError,
	CloudErrorBody: &api.CloudErrorBody{
		Code:    api.CloudErrorCodeInternalServerError,
		Message: "Deployment failed.",
	},
}

// parseInstallationFailure analyzes installation logs and returns a structured error
// Reuses the failure detection patterns from pkg/hive/failure
func parseInstallationFailure(installLog string) error {
	if installLog == "" {
		return genericErr
	}

	// Check each known failure pattern from Hive
	for _, reason := range failure.Reasons {
		for _, searchRegex := range reason.SearchRegexes {
			if searchRegex.MatchString(installLog) {
				return handleKnownFailure(reason, installLog)
			}
		}
	}

	// No known pattern matched - return generic error
	return genericErr
}

// handleKnownFailure processes a known failure type and returns appropriate CloudError
func handleKnownFailure(reason failure.InstallFailingReason, installLog string) error {
	switch reason.Reason {
	case "AzureKeyBasedAuthenticationNotPermitted":
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: reason.Message,
				Details: []api.CloudErrorBody{
					{
						Message: "Cluster creation failed because key based authentication has been disabled on the ARO cluster storage account, most likely by an Azure Policy modify effect. Please ensure that any policies in your Azure subscription do not mutate this property on the storage accounts provisioned during ARO cluster deployment. See https://learn.microsoft.com/en-us/azure/governance/policy/concepts/effect-modify for more details.",
					},
				},
			},
		}

	case "AzureRequestDisallowedByPolicy":
		armError, err := parseDeploymentFailedJson(installLog)
		if err != nil {
			return genericErr
		}
		return wrapArmError(reason.Message, *armError)

	case "AzureInvalidTemplateDeployment":
		armError, err := parseDeploymentFailedJson(installLog)
		if err != nil {
			return genericErr
		}
		return wrapArmError(reason.Message, *armError)

	case "AzureZonalAllocationFailed":
		armError, err := parseDeploymentFailedJson(installLog)
		if err != nil {
			return genericErr
		}
		return wrapArmError(reason.Message, *armError)

	case "AzureOSProvisioningTimedOut":
		armError, err := parseDeploymentFailedJson(installLog)
		if err != nil {
			return genericErr
		}
		return wrapArmError(reason.Message, *armError)

	default:
		return genericErr
	}
}

// parseDeploymentFailedJson extracts ARM error JSON from installer logs
func parseDeploymentFailedJson(installLog string) (*mgmtfeatures.ErrorResponse, error) {
	regex := regexp.MustCompile(`level=error msg=400: DeploymentFailed: : Deployment failed. Details: : : (\{.*\})`)
	matches := regex.FindStringSubmatch(installLog)
	if len(matches) < 2 {
		return nil, &api.CloudError{
			StatusCode: http.StatusInternalServerError,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeInternalServerError,
				Message: "Failed to parse deployment error details.",
			},
		}
	}

	rawJSON := matches[1]
	armResponse := &mgmtfeatures.ErrorResponse{}
	if err := json.Unmarshal([]byte(rawJSON), armResponse); err != nil {
		return nil, &api.CloudError{
			StatusCode: http.StatusInternalServerError,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeInternalServerError,
				Message: "Failed to parse deployment error JSON.",
			},
		}
	}

	return armResponse, nil
}

// wrapArmError creates a CloudError from an ARM error response
func wrapArmError(errorMessage string, armError mgmtfeatures.ErrorResponse) *api.CloudError {
	var details []api.CloudErrorBody
	if armError.Details != nil {
		details = make([]api.CloudErrorBody, len(*armError.Details))
		for i, detail := range *armError.Details {
			details[i] = errorResponseToCloudErrorBody(detail)
		}
	}

	return &api.CloudError{
		StatusCode: http.StatusBadRequest,
		CloudErrorBody: &api.CloudErrorBody{
			Code:    api.CloudErrorCodeDeploymentFailed,
			Message: errorMessage,
			Details: details,
		},
	}
}

// errorResponseToCloudErrorBody converts ARM error to CloudErrorBody
func errorResponseToCloudErrorBody(errorResponse mgmtfeatures.ErrorResponse) api.CloudErrorBody {
	body := api.CloudErrorBody{}

	if errorResponse.Code != nil {
		body.Code = *errorResponse.Code
	}
	if errorResponse.Message != nil {
		body.Message = *errorResponse.Message
	}
	if errorResponse.Target != nil {
		body.Target = *errorResponse.Target
	}

	return body
}
