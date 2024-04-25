package armkeyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type VaultsClient interface {
	CheckNameAvailability(ctx context.Context, vaultName armkeyvault.VaultCheckNameAvailabilityParameters, options *armkeyvault.VaultsClientCheckNameAvailabilityOptions) (armkeyvault.VaultsClientCheckNameAvailabilityResponse, error)
}

type vaultsClient struct {
	*armkeyvault.VaultsClient
}

var _ VaultsClient = &vaultsClient{}

func NewVaultsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (VaultsClient, error) {
	client, err := armkeyvault.NewVaultsClient(subscriptionID, credential, options)
	return vaultsClient{
		VaultsClient: client,
	}, err
}
