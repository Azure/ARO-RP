package failure

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	corev1 "k8s.io/api/core/v1"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	hivev1 "github.com/openshift/hive/apis/hive/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

var genericErr = &api.CloudError{
	StatusCode: http.StatusInternalServerError,
	CloudErrorBody: &api.CloudErrorBody{
		Code:    api.CloudErrorCodeInternalServerError,
		Message: "Deployment failed.",
	},
}

func HandleProvisionFailed(ctx context.Context, cd *hivev1.ClusterDeployment, cond hivev1.ClusterDeploymentCondition, installLog *string) error {
	if cond.Status != corev1.ConditionTrue {
		return nil
	}

	switch cond.Reason {
	case AzureKeyBasedAuthenticationNotPermitted.Reason:
		// Construct the error directly since this error appears in install logs as an HTTP error response
		// rather than a more easily parsable ARM error.
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: AzureKeyBasedAuthenticationNotPermitted.Message,
				Details: []api.CloudErrorBody{
					{
						Message: "Cluster creation failed because key based authentication has been disabled on the ARO cluster storage account, most likely by an Azure Policy modify effect. Please ensure that any policies in your Azure subscription do not mutate this property on the storage accounts provisioned during ARO cluster deployment. See https://learn.microsoft.com/en-us/azure/governance/policy/concepts/effect-modify for more details.",
					},
				},
			},
		}
	case AzureRequestDisallowedByPolicy.Reason:
		armError, err := parseDeploymentFailedJson(*installLog)
		if err != nil {
			return err
		}

		return wrapArmError(
			AzureRequestDisallowedByPolicy.Message,
			*armError,
		)
	case AzureInvalidTemplateDeployment.Reason:
		armError, err := parseDeploymentFailedJson(*installLog)
		if err != nil {
			return err
		}

		return wrapArmError(
			AzureInvalidTemplateDeployment.Message,
			*armError,
		)
	case AzureZonalAllocationFailed.Reason:
		armError, err := parseDeploymentFailedJson(*installLog)
		if err != nil {
			return err
		}

		return wrapArmError(
			AzureZonalAllocationFailed.Message,
			*armError,
		)
	default:
		return genericErr
	}
}

func parseDeploymentFailedJson(installLog string) (*mgmtfeatures.ErrorResponse, error) {
	regex := regexp.MustCompile(`level=error msg=400: DeploymentFailed: : Deployment failed. Details: : : (\{.*\})`)
	rawJson := regex.FindStringSubmatch(installLog)[1]

	armResponse := &mgmtfeatures.ErrorResponse{}
	if err := json.Unmarshal([]byte(rawJson), armResponse); err != nil {
		return nil, err
	}
	return armResponse, nil
}

func wrapArmError(errorMessage string, armError mgmtfeatures.ErrorResponse) *api.CloudError {
	details := make([]api.CloudErrorBody, len(*armError.Details))
	for i, detail := range *armError.Details {
		details[i] = errorResponseToCloudErrorBody(detail)
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

func errorResponseToCloudErrorBody(errorResponse mgmtfeatures.ErrorResponse) api.CloudErrorBody {
	body := api.CloudErrorBody{
		Code:    *errorResponse.Code,
		Message: *errorResponse.Message,
	}

	if errorResponse.Target != nil {
		body.Target = *errorResponse.Target
	}

	return body
}
