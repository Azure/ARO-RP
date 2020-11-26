package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
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
func NewTokensClient(environment *azure.Environment, subscriptionID string, authorizer autorest.Authorizer) TokensClient {
	client := mgmtcontainerregistry.NewTokensClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &tokensClient{
		TokensClient: client,
	}
}
