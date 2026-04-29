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
	subscriptionID, tenantID string) error {

	fpAuthorizer, err := environment.FPAuthorizer(
		tenantID, nil,
		environment.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}

	providersClient := features.NewProvidersClient(
		azEnv, subscriptionID, fpAuthorizer)

	providers, err := providersClient.List(ctx, nil, "")
	if err != nil {
		return err
	}

	for _, provider := range providers {
		if *provider.Namespace == "Microsoft.Compute" {
			for _, resourceType := range *provider.ResourceTypes {
				if *resourceType.ResourceType == "encryptionAtHost" {
					return nil
				}
			}
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
	}
	return nil
}
