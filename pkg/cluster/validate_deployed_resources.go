package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	utilarm "github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/sirupsen/logrus"
)

type propMismatch struct {
	Resource string      `json:"resource"`
	Property string      `json:"property"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
}

func (m *manager) validateDeployedResources(ctx context.Context) error {
	var allMismatches []propMismatch

	mismatches, err := m.validateManagedStorageAccounts(ctx)
	if err != nil {
		return err
	}
	if len(mismatches) > 0 {
		allMismatches = append(allMismatches, mismatches...)
	}

	if len(allMismatches) > 0 {
		mismatchJSON, _ := json.Marshal(allMismatches)
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: fmt.Sprintf("Deployment succeeded but post-deployment validation detected unexpected mutation, check policy. Properties mismatch: %s", string(mismatchJSON)),
			},
		}
	}

	return nil
}

func (m *manager) validateManagedStorageAccounts(ctx context.Context) ([]propMismatch, error) {
	var allMismatches []propMismatch
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	expectedProperties := map[string]interface{}{
		"publicNetworkAccess": "Enabled",
	}
	storageAccountNames := []string{
		"cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix,
		m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName,
	}

	for _, storageAccountName := range storageAccountNames {
		m.log.Debugf("validating storage account: %s", storageAccountName)

		var propsJSON []byte
		err := utilarm.RetryableWith(ctx, func(err error) bool {
			return azureerrors.IsStatusForbiddenError(err) || azureerrors.IsRetryableError(err)
		}, func() error {
			var getErr error
			storageAccount, getErr := m.storage.GetProperties(ctx, resourceGroup, storageAccountName)
			if getErr != nil {
				return getErr
			}
			if storageAccount.Properties != nil {
				propsJSON, getErr = json.Marshal(storageAccount.Properties)
			}
			return getErr
		}, m.log, fmt.Sprintf("fetching storage account properties for %s", storageAccountName), &utilarm.RetryOptions{Steps: utilarm.RBACPropagationRetrySteps})
		if err != nil {
			return nil, fmt.Errorf("unable to fetch storage account properties for %s: %w", storageAccountName, err)
		}

		mismatches, err := ValidateDeployedResourceState(ctx, m.log, propsJSON, expectedProperties, storageAccountName)
		if err != nil {
			return nil, err
		}
		if len(mismatches) > 0 {
			allMismatches = append(allMismatches, mismatches...)
		}
	}

	return allMismatches, nil
}

func ValidateDeployedResourceState(ctx context.Context, log *logrus.Entry, propsJSON []byte, expectedProperties map[string]interface{}, resourceName string) ([]propMismatch, error) {
	if propsJSON == nil {
		return nil, nil
	}
	propsMap := make(map[string]interface{})
	if err := json.Unmarshal(propsJSON, &propsMap); err != nil {
		return nil, fmt.Errorf("unable to unmarshal properties for %s: %w", resourceName, err)
	}

	var mismatches []propMismatch
	validatePropertiesRecursive(propsMap, expectedProperties, resourceName, "", &mismatches)
	return mismatches, nil
}

func validatePropertiesRecursive(actualProps map[string]interface{}, expectedProps map[string]interface{}, resourceName string, prefix string, mismatches *[]propMismatch) {
	for propName, expectedValue := range expectedProps {
		actualValue, exists := actualProps[propName]
		if !exists {
			continue
		}

		propPath := propName
		if prefix != "" {
			propPath = prefix + "." + propName
		}

		expectedMap, expectedIsMap := expectedValue.(map[string]interface{})
		actualMap, actualIsMap := actualValue.(map[string]interface{})

		if expectedIsMap && actualIsMap {
			validatePropertiesRecursive(actualMap, expectedMap, resourceName, propPath, mismatches)
		} else if actualValue != expectedValue {
			*mismatches = append(*mismatches, propMismatch{
				Resource: resourceName,
				Property: propPath,
				Expected: expectedValue,
				Actual:   actualValue,
			})
		}
	}
}
