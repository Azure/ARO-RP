package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
)

func validateEncryptionAtHostFeature(
	ctx context.Context,
	azEnv *azureclient.AROEnvironment,
	environment env.Interface,
	subscriptionID, tenantID string,
) error {
	fpAuthorizer, err := environment.FPAuthorizer(
		tenantID, nil,
		environment.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}

	resourcesClient := features.NewResourcesClient(
		azEnv, subscriptionID, fpAuthorizer)

	resourceID := fmt.Sprintf(
		"/subscriptions/%s/providers/Microsoft.Features"+
			"/features/EncryptionAtHost",
		subscriptionID)

	feature, err := resourcesClient.GetByID(
		ctx, resourceID, "2021-07-01")
	if err != nil {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf(
				"Microsoft.Compute/EncryptionAtHost"+
					" is not registered for"+
					" subscription %s.",
				subscriptionID))
	}

	if feature.Properties == nil {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf(
				"Microsoft.Compute/EncryptionAtHost"+
					" is not registered for"+
					" subscription %s.",
				subscriptionID))
	}

	properties, ok := feature.Properties.(map[string]interface{})
	if !ok || properties["state"] != "Registered" {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf(
				"Microsoft.Compute/EncryptionAtHost"+
					" is not registered for"+
					" subscription %s.",
				subscriptionID))
	}

	return nil
}
