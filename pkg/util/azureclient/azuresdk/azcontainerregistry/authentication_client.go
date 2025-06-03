package azcontainerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"
)

type AuthenticationClient interface {
	ExchangeAADAccessTokenForACRRefreshToken(ctx context.Context, grantType azcontainerregistry.PostContentSchemaGrantType, service string, options *azcontainerregistry.AuthenticationClientExchangeAADAccessTokenForACRRefreshTokenOptions) (azcontainerregistry.AuthenticationClientExchangeAADAccessTokenForACRRefreshTokenResponse, error)
}

type ArmAuthenticationClient struct {
	*azcontainerregistry.AuthenticationClient
}

var _ AuthenticationClient = &ArmAuthenticationClient{}

func NewAuthenticationClient(endpoint string, options azcore.ClientOptions) (AuthenticationClient, error) {
	clientOptions := azcontainerregistry.AuthenticationClientOptions{
		ClientOptions: options,
	}
	_client, err := azcontainerregistry.NewAuthenticationClient(endpoint, &clientOptions)
	return ArmAuthenticationClient{
		AuthenticationClient: _client,
	}, err
}
