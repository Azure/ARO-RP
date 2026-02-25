package armcontainerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	armcontainerregistry "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2"
)

type TokensClientAddons interface {
	CreateAndWait(ctx context.Context, resourceGroupName string, registryName string, tokenName string, tokenCreateParameters armcontainerregistry.Token) (*armcontainerregistry.TokensClientCreateResponse, error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, registryName string, tokenName string) error
	GetTokenProperties(ctx context.Context, resourceGroupName string, registryName string, tokenName string) (*armcontainerregistry.TokenProperties, error)
}

func (c *tokensClient) CreateAndWait(ctx context.Context, resourceGroupName string, registryName string, tokenName string, tokenCreateParameters armcontainerregistry.Token) (*armcontainerregistry.TokensClientCreateResponse, error) {
	poller, err := c.BeginCreate(ctx, resourceGroupName, registryName, tokenName, tokenCreateParameters, nil)
	if err != nil {
		return nil, err
	}

	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *tokensClient) DeleteAndWait(ctx context.Context, resourceGroupName string, registryName string, tokenName string) error {
	poller, err := c.BeginDelete(ctx, resourceGroupName, registryName, tokenName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *tokensClient) GetTokenProperties(ctx context.Context, resourceGroupName string, registryName string, tokenName string) (*armcontainerregistry.TokenProperties, error) {
	resp, err := c.Get(ctx, resourceGroupName, registryName, tokenName, nil)
	if err != nil {
		return nil, err
	}
	return resp.Properties, nil
}
