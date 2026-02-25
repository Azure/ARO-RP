package armcontainerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcontainerregistry "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2"
)

type TokensClient interface {
	TokensClientAddons
}

type tokensClient struct {
	*armcontainerregistry.TokensClient
}

var _ TokensClient = &tokensClient{}

// NewTokensClient creates a new TokensClient
func NewTokensClient(subscriptionId string, credential azcore.TokenCredential, options *arm.ClientOptions) (TokensClient, error) {
	clientFactory, err := armcontainerregistry.NewClientFactory(subscriptionId, credential, options)
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewTokensClient()

	return &tokensClient{
		TokensClient: client,
	}, nil
}
