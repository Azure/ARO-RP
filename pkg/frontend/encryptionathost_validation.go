package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/go-autorest/autorest"

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

	providersClient := features.NewProvidersClient(
		azEnv, subscriptionID, fpAuthorizer)

	top := int32(0)
	providers, err := providersClient.List(ctx, &top, "")
	if err != nil {
		if detailed, ok := err.(autorest.DetailedError); ok {
			if detailed.StatusCode == http.StatusNotFound {
				return api.NewCloudError(
					http.StatusBadRequest,
					api.CloudErrorCodeInvalidParameter,
					"encryptionAtHost",
					fmt.Sprintf(
						"Microsoft.Compute/EncryptionAtHost"+
							" is not registered for"+
							" subscription %s.",
						subscriptionID))
			}
		}
		return err
	}

	for _, provider := range providers {
		if provider.Namespace != nil &&
			*provider.Namespace == "Microsoft.Compute" &&
			provider.RegistrationState != nil &&
			*provider.RegistrationState == "Registered" {
			return nil
		}
	}

	return api.NewCloudError(
		http.StatusBadRequest,
		api.CloudErrorCodeInvalidParameter,
		"encryptionAtHost",
		fmt.Sprintf(
			"Microsoft.Compute/EncryptionAtHost"+
				" is not registered for"+
				" subscription %s.",
			subscriptionID))
}
