package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/$GOPACKAGE VaultsClient
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

// VaultsClient is a minimal interface for azure VaultsClient
type VaultsClient interface {
	VaultsClientAddons
}

type vaultsClient struct {
	keyvault.VaultsClient
}

var _ VaultsClient = &vaultsClient{}

// NewVaultsClient creates a new VaultsClient
func NewVaultsClient(subscriptionID string, authorizer autorest.Authorizer) VaultsClient {
	client := keyvault.NewVaultsClient(subscriptionID)
	client.Authorizer = authorizer

	return &vaultsClient{
		VaultsClient: client,
	}
}
