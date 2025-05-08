package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// TokensClient is a minimal interface for azure TokensClient
type TokensClient interface {
	TokensAddons
}

type tokensClient struct {
	mgmtcontainerregistry.TokensClient
}

var _ TokensClient = &tokensClient{}

// NewTokensClient creates a new TokensClient
func NewTokensClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) TokensClient {
	client := mgmtcontainerregistry.NewTokensClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &tokensClient{
		TokensClient: client,
	}
}
