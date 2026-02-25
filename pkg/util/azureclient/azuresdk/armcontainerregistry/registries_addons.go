package armcontainerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	armcontainerregistry "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2"
)

type RegistriesClientAddons interface {
	GenerateCredentialsAndWait(ctx context.Context, resourceGroupName string, registryName string, generateCredentialsParameters armcontainerregistry.GenerateCredentialsParameters) (*armcontainerregistry.GenerateCredentialsResult, error)
}

func (c *registriesClient) GenerateCredentialsAndWait(ctx context.Context, resourceGroupName string, registryName string, generateCredentialsParameters armcontainerregistry.GenerateCredentialsParameters) (*armcontainerregistry.GenerateCredentialsResult, error) {
	poller, err := c.BeginGenerateCredentials(ctx, resourceGroupName, registryName, generateCredentialsParameters, nil)
	if err != nil {
		return nil, err
	}

	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &res.GenerateCredentialsResult, nil
}
