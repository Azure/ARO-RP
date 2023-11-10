package armkeyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	sdkkeyvault "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
)

type VaultsClient interface {
	CheckNameAvailability(ctx context.Context, vaultName sdkkeyvault.VaultCheckNameAvailabilityParameters, options *sdkkeyvault.VaultsClientCheckNameAvailabilityOptions) (sdkkeyvault.VaultsClientCheckNameAvailabilityResponse, error)
}

type vaultsClient struct {
	*sdkkeyvault.VaultsClient
}

var _ VaultsClient = &vaultsClient{}

func NewVaultsClient(subscriptionID string, credential azcore.TokenCredential, options *azidentity.EnvironmentCredentialOptions) (VaultsClient, error) {
	clientOption := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: options.Cloud,
		},
	}
	client, err := sdkkeyvault.NewVaultsClient(subscriptionID, credential, clientOption)
	return vaultsClient{
		VaultsClient: client,
	}, err
}
