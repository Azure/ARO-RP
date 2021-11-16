package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

// VaultsClient is a minimal interface for azure VaultsClient
type VaultsClient interface {
	CheckNameAvailability(ctx context.Context, vaultName mgmtkeyvault.VaultCheckNameAvailabilityParameters) (result mgmtkeyvault.CheckNameAvailabilityResult, err error)
}

type vaultsClient struct {
	mgmtkeyvault.VaultsClient
}

var _ VaultsClient = &vaultsClient{}

// NewDisksClient creates a new DisksClient
func NewVaultsClient(subscriptionID string, authorizer autorest.Authorizer) VaultsClient {
	client := mgmtkeyvault.NewVaultsClient(subscriptionID)
	client.Authorizer = authorizer

	return &vaultsClient{
		VaultsClient: client,
	}
}
