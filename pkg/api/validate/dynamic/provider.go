package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (dv *dynamic) ValidateProviders(ctx context.Context) error {
	dv.log.Print("ValidateProviders")

	providers, err := dv.providers.List(ctx, nil, "")
	if err != nil {
		return err
	}

	providerMap := make(map[string]mgmtresources.Provider, len(providers))

	for _, provider := range providers {
		providerMap[*provider.Namespace] = provider
	}

	for _, provider := range []string{
		"Microsoft.Authorization",
		"Microsoft.Compute",
		"Microsoft.Network",
		"Microsoft.Storage",
	} {
		if providerMap[provider].RegistrationState == nil ||
			*providerMap[provider].RegistrationState != "Registered" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorResourceProviderNotRegistered, "", "The resource provider '%s' is not registered.", provider)
		}
	}

	return nil
}
