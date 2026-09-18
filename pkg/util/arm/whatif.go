package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
)

func validateDeploymentWithWhatIf(ctx context.Context, log *logrus.Entry, deployments features.DeploymentsClient, resourceGroupName string, deploymentName string, template *Template, resourcesToValidate map[string]map[string]interface{}) error {
	log.Printf("validating %s deployment with what-if", deploymentName)

	whatIfParams := mgmtfeatures.DeploymentWhatIf{
		Properties: &mgmtfeatures.DeploymentWhatIfProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	}

	whatIfResult, err := deployments.WhatIfAndWait(ctx, resourceGroupName, deploymentName, whatIfParams)
	if err != nil {
		return err
	}

	if whatIfResult.WhatIfOperationProperties == nil || whatIfResult.Changes == nil {
		return nil
	}

	var allMismatches []map[string]interface{}
	for _, change := range *whatIfResult.Changes {
		if change.ResourceID != nil {
			resourceConfig, err := findMatchingResourceConfig(*change.ResourceID, resourcesToValidate)
			if err != nil {
				return err
			}
			if resourceConfig != nil && change.Delta != nil {
				mismatches := collectPropertyChanges(*change.Delta, change.ResourceID, resourceConfig)
				allMismatches = append(allMismatches, mismatches...)
			}
		}
	}

	if len(allMismatches) > 0 {
		mismatchJSON, _ := json.Marshal(allMismatches)
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: fmt.Sprintf("Deployment succeeded but post-deployment validation detected unexpected mutations, likely due to Azure policies. Details: %s", string(mismatchJSON)),
			},
		}
	}

	return nil
}

func findMatchingResourceConfig(resourceID string, resourcesToValidate map[string]map[string]interface{}) (map[string]interface{}, error) {
	parsed, err := arm.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}
	extractedName := parsed.Name
	if config, exists := resourcesToValidate[extractedName]; exists {
		expectedType, ok := config["resourceType"].(string)
		if ok && strings.Contains(parsed.ResourceType.String(), strings.ToLower(expectedType)) {
			return config["expectedProperties"].(map[string]interface{}), nil
		}
	}
	return nil, nil
}

func collectPropertyChanges(propertyChanges []mgmtfeatures.WhatIfPropertyChange, resourceID *string, expectedProperties map[string]interface{}) []map[string]interface{} {
	var mismatches []map[string]interface{}

	for _, propChange := range propertyChanges {
		if propChange.Path != nil {
			path := *propChange.Path

			if expectedVal, ok := expectedProperties[path]; ok {
				if propChange.Before != nil && propChange.Before != expectedVal {
					resID := ""
					if resourceID != nil {
						resID = *resourceID
					}
					mismatch := map[string]interface{}{
						"resource": resID,
						"property": path,
						"expected": expectedVal,
						"actual":   propChange.Before,
					}
					mismatches = append(mismatches, mismatch)
				}
			}
		}

		if propChange.Children != nil {
			childMismatches := collectPropertyChanges(*propChange.Children, resourceID, expectedProperties)
			mismatches = append(mismatches, childMismatches...)
		}
	}

	return mismatches
}

func BuildStorageAccountsValidationMap(storageAccountNames ...string) map[string]map[string]interface{} {
	resourcesToValidate := make(map[string]map[string]interface{})
	expectedProperties := map[string]interface{}{
		"/properties/publicNetworkAccess": "Enabled",
	}
	for _, accountName := range storageAccountNames {
		if accountName != "" {
			resourcesToValidate[accountName] = map[string]interface{}{
				"resourceType":       "Microsoft.Storage/storageAccounts",
				"expectedProperties": expectedProperties,
			}
		}
	}
	return resourcesToValidate
}
