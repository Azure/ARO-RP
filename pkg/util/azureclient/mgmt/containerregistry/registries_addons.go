package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	azcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
)

// RegistriesAddons contains addons for RegistriesClient
type RegistriesAddons interface {
	GenerateCredentials(ctx context.Context, resourceGroupName string, registryName string, generateCredentialsParameters azcontainerregistry.GenerateCredentialsParameters) (result azcontainerregistry.GenerateCredentialsResult, err error)
}

func (r *registriesClient) GenerateCredentials(ctx context.Context, resourceGroupName string, registryName string, generateCredentialsParameters azcontainerregistry.GenerateCredentialsParameters) (azcontainerregistry.GenerateCredentialsResult, error) {
	future, err := r.RegistriesClient.GenerateCredentials(ctx, resourceGroupName, registryName, generateCredentialsParameters)
	if err != nil {
		return azcontainerregistry.GenerateCredentialsResult{}, err
	}

	err = future.WaitForCompletionRef(ctx, r.Client)
	if err != nil {
		return azcontainerregistry.GenerateCredentialsResult{}, err
	}
	return r.RegistriesClient.GenerateCredentialsResponder(future.Response())
}
