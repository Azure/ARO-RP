package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
)

// RegistriesAddons contains addons for RegistriesClient
type RegistriesAddons interface {
	GenerateCredentials(ctx context.Context, resourceGroupName string, registryName string, generateCredentialsParameters mgmtcontainerregistry.GenerateCredentialsParameters) (result mgmtcontainerregistry.GenerateCredentialsResult, err error)
}

func (r *registriesClient) GenerateCredentials(ctx context.Context, resourceGroupName string, registryName string, generateCredentialsParameters mgmtcontainerregistry.GenerateCredentialsParameters) (mgmtcontainerregistry.GenerateCredentialsResult, error) {
	future, err := r.RegistriesClient.GenerateCredentials(ctx, resourceGroupName, registryName, generateCredentialsParameters)
	if err != nil {
		return mgmtcontainerregistry.GenerateCredentialsResult{}, err
	}

	err = future.WaitForCompletionRef(ctx, r.Client)
	if err != nil {
		return mgmtcontainerregistry.GenerateCredentialsResult{}, err
	}
	return r.RegistriesClient.GenerateCredentialsResponder(future.Response())
}
