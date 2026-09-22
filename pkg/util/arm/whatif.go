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

func ValidateDeploymentWithWhatIf(ctx context.Context, log *logrus.Entry, deployments features.DeploymentsClient, resourceGroupName string, deploymentName string, template *Template, resourcesToValidate map[string]map[string]interface{}) error {
	log.Printf("validating %s deployment with what-if", deploymentName)

	whatIfParams := mgmtfeatures.DeploymentWhatIf{
		Properties: &mgmtfeatures.DeploymentWhatIfProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	}

	whatIfResult, err := deployments.WhatIfAndWait(ctx, resourceGroupName, deploymentName, whatIfParams)
	if err != nil {
		log.Errorf("ValidateDeploymentWithWhatIf failed with error: %v", err)
		return nil
	}

	if whatIfResult.WhatIfOperationProperties == nil || whatIfResult.Changes == nil {
		log.Infof("WhatIf result has no changes or operation properties for deployment: %s", deploymentName)
		return nil
	}

	var allMismatches []map[string]interface{}
	for _, change := range *whatIfResult.Changes {
		if change.ResourceID != nil {
			parsedResourceID, err := arm.ParseResourceID(*change.ResourceID)
			if err != nil {
				log.Errorf("ParseResourceID failed with error: %v", err)
				continue
			}
			resourceConfig, err := findMatchingResourceConfig(parsedResourceID, resourcesToValidate)
			if err != nil {
				continue
			}
			if resourceConfig != nil && change.Delta != nil {
				mismatches := collectPropertyChanges(log, *change.Delta, parsedResourceID, resourceConfig)
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
				Message: "Deployment failed.",
				Details: []api.CloudErrorBody{
					{
						Message: fmt.Sprintf("Unexpected property mutations detected, likely due to Azure policies. Details: %s", string(mismatchJSON)),
					},
				},
			},
		}
	}

	return nil
}

func findMatchingResourceConfig(parsedResourceID *arm.ResourceID, resourcesToValidate map[string]map[string]interface{}) (map[string]interface{}, error) {
	extractedName := strings.ToLower(parsedResourceID.Name)
	if config, exists := resourcesToValidate[extractedName]; exists {
		expectedType, ok := config["resourceType"].(string)
		if ok && strings.Contains(strings.ToLower(parsedResourceID.ResourceType.String()), strings.ToLower(expectedType)) {
			return config["expectedProperties"].(map[string]interface{}), nil
		}
	}
	return nil, nil
}

func collectPropertyChanges(log *logrus.Entry, propertyChanges []mgmtfeatures.WhatIfPropertyChange, parsedResourceID *arm.ResourceID, expectedProperties map[string]interface{}) []map[string]interface{} {
	var mismatches []map[string]interface{}
	for _, propChange := range propertyChanges {
		if propChange.Path != nil {
			path := normalizePath(*propChange.Path)

			log.Infof("Checking property change for path: %s, value %v, for resource: %s", path, propChange.Before, parsedResourceID.String())
			if expectedVal, ok := expectedProperties[path]; ok {
				if propChange.Before != nil && propChange.Before != expectedVal {
					mismatch := map[string]interface{}{
						"resourceName":  parsedResourceID.Name,
						"resourceType":  parsedResourceID.ResourceType.String(),
						"resourceGroup": parsedResourceID.ResourceGroupName,
						"property":      path,
						"expected":      expectedVal,
						"actual":        propChange.Before,
					}
					mismatches = append(mismatches, mismatch)
				}
			}
		}

		if propChange.Children != nil {
			childMismatches := collectPropertyChanges(log, *propChange.Children, parsedResourceID, expectedProperties)
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
			resourcesToValidate[strings.ToLower(accountName)] = map[string]interface{}{
				"resourceType":       "Microsoft.Storage/storageAccounts",
				"expectedProperties": expectedProperties,
			}
		}
	}
	return resourcesToValidate
}

func normalizePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + strings.ReplaceAll(path, ".", "/")
	}
	return path
}
