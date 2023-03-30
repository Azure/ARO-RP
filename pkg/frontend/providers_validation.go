package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
)

type ProvidersValidator interface {
	ValidateProviders(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string) error
}

type providersValidator struct{}

var requiredResourceProviders = []string{
	"Microsoft.Authorization",
	"Microsoft.Compute",
	"Microsoft.Network",
	"Microsoft.Storage",
}

func (p providersValidator) ValidateProviders(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string) error {
	fpAuthorizer, err := environment.FPAuthorizer(tenantID, environment.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	providersClient := features.NewProvidersClient(azEnv, subscriptionID, fpAuthorizer)

	return validateProviders(ctx, providersClient)
}

func validateProviders(ctx context.Context, providersClient features.ProvidersClient) error {
	providers, err := providersClient.List(ctx, nil, "")
	if err != nil {
		return err
	}

	providerMap := make(map[string]mgmtfeatures.Provider, len(providers))

	for _, provider := range providers {
		providerMap[*provider.Namespace] = provider
	}

	for _, provider := range requiredResourceProviders {
		if providerMap[provider].RegistrationState == nil ||
			*providerMap[provider].RegistrationState != "Registered" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceProviderNotRegistered, "", "The resource provider '%s' is not registered.", provider)
		}
	}

	return nil
}
