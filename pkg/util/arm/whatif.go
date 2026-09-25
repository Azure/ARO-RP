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

type Mismatch struct {
	ResourceName  string              `json:"resourceName"`
	ResourceType  string              `json:"resourceType"`
	ResourceGroup string              `json:"resourceGroup"`
	Property      string              `json:"property"`
	Expected      interface{}         `json:"expected"`
	Actual        interface{}         `json:"actual"`
	Policies      []PolicyAttribution `json:"policies,omitempty"`
}

type PolicyAttribution struct {
	Field            string     `json:"field"`
	Result           string     `json:"result,omitempty"`
	Effect           string     `json:"effect,omitempty"`
	Values           []string   `json:"values,omitempty"`
	PolicyDefinition *PolicyRef `json:"policyDefinition,omitempty"`
	PolicyAssignment *PolicyRef `json:"policyAssignment,omitempty"`
}

type PolicyRef struct {
	Name              string `json:"name,omitempty"`
	SubscriptionID    string `json:"subscriptionId,omitempty"`
	ResourceGroupName string `json:"resourceGroupName,omitempty"`
}

func ValidateDeploymentWithWhatIf(ctx context.Context, log *logrus.Entry, deployments features.DeploymentsClient, resourceGroupName string, deploymentName string, template *Template, resourcesToValidate map[string]map[string]interface{}) []Mismatch {
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

	var allMismatches []Mismatch
	for _, change := range *whatIfResult.Changes {
		if change.ResourceID != nil {
			parsedResourceID, err := arm.ParseResourceID(*change.ResourceID)
			if err != nil {
				log.Errorf("ParseResourceID failed with error: %v", err)
				continue
			}
			resourceConfig := findMatchingResourceConfig(parsedResourceID, resourcesToValidate)
			if resourceConfig != nil && change.Delta != nil {
				mismatches := collectPropertyChanges(log, *change.Delta, parsedResourceID, resourceConfig)
				allMismatches = append(allMismatches, mismatches...)
			}
		}
	}

	return allMismatches
}

func MismatchesToCloudError(mismatches []Mismatch) error {
	if len(mismatches) == 0 {
		return nil
	}
	mismatchJSON, _ := json.Marshal(mismatches)
	return &api.CloudError{
		StatusCode: http.StatusBadRequest,
		CloudErrorBody: &api.CloudErrorBody{
			Code:    api.CloudErrorCodeDeploymentFailed,
			Message: "Deployment failed.",
			Details: []api.CloudErrorBody{
				{
					Message: fmt.Sprintf("%s%s", mismatchDetailsPrefix, string(mismatchJSON)),
				},
			},
		},
	}
}

func findMatchingResourceConfig(parsedResourceID *arm.ResourceID, resourcesToValidate map[string]map[string]interface{}) map[string]interface{} {
	config, exists := resourcesToValidate[strings.ToLower(parsedResourceID.Name)]
	if !exists {
		return nil
	}
	expectedType, ok := config["resourceType"].(string)
	if !ok || !strings.EqualFold(parsedResourceID.ResourceType.String(), expectedType) {
		return nil
	}
	expectedProperties, ok := config["expectedProperties"].(map[string]interface{})
	if !ok {
		return nil
	}
	return expectedProperties
}

func collectPropertyChanges(log *logrus.Entry, propertyChanges []mgmtfeatures.WhatIfPropertyChange, parsedResourceID *arm.ResourceID, expectedProperties map[string]interface{}) []Mismatch {
	var mismatches []Mismatch
	for _, propChange := range propertyChanges {
		if propChange.Path != nil {
			path := normalizePath(*propChange.Path)

			log.Infof("Checking property change for path: %s, value %v, for resource: %s", path, propChange.Before, parsedResourceID.String())
			if expectedVal, ok := expectedProperties[path]; ok {
				if propChange.Before != expectedVal {
					mismatches = append(mismatches, Mismatch{
						ResourceName:  parsedResourceID.Name,
						ResourceType:  parsedResourceID.ResourceType.String(),
						ResourceGroup: parsedResourceID.ResourceGroupName,
						Property:      path,
						Expected:      expectedVal,
						Actual:        propChange.Before,
					})
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
