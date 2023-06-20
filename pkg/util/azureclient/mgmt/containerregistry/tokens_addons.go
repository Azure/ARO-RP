package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
)

// TokensAddons contains addons for TokensClient
type TokensAddons interface {
	CreateAndWait(ctx context.Context, resourceGroupName string, registryName string, tokenName string, tokenCreateParameters mgmtcontainerregistry.Token) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, registryName string, tokenName string) error
	GetTokenProperties(ctx context.Context, resourceGroupName, registryName, tokenName string) (mgmtcontainerregistry.TokenProperties, error)
}

func (t *tokensClient) CreateAndWait(ctx context.Context, resourceGroupName string, registryName string, tokenName string, tokenCreateParameters mgmtcontainerregistry.Token) error {
	future, err := t.TokensClient.Create(ctx, resourceGroupName, registryName, tokenName, tokenCreateParameters)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, t.Client)
}

func (t *tokensClient) DeleteAndWait(ctx context.Context, resourceGroupName string, registryName string, tokenName string) error {
	future, err := t.TokensClient.Delete(ctx, resourceGroupName, registryName, tokenName)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, t.Client)
}

func (t *tokensClient) GetTokenProperties(ctx context.Context, resourceGroupName, registryName, tokenName string) (mgmtcontainerregistry.TokenProperties, error) {
	var token mgmtcontainerregistry.Token
	token, err := t.TokensClient.Get(ctx, resourceGroupName, registryName, tokenName)
	if err != nil {
		return mgmtcontainerregistry.TokenProperties{}, err
	}
	return *token.TokenProperties, nil
}
