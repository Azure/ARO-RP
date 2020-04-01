package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

// VaultsClient is a minimal interface for azure VaultsClient
type VaultsClient interface {
	VaultsClientAddons
}

type vaultsClient struct {
	mgmtkeyvault.VaultsClient
}

var _ VaultsClient = &vaultsClient{}

// NewVaultsClient creates a new VaultsClient
func NewVaultsClient(subscriptionID string, authorizer autorest.Authorizer) VaultsClient {
	client := mgmtkeyvault.NewVaultsClient(subscriptionID)
	client.Authorizer = authorizer

	return &vaultsClient{
		VaultsClient: client,
	}
}
