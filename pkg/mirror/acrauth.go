package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"time"

	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"

	azcorepolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	sdkcontainerregistry "github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcontainerregistry"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

const (
	tokenExpirationTime = 3 * time.Hour
	expirationBuffer    = 15 * time.Minute
	rotateAfter         = tokenExpirationTime - expirationBuffer
)

type AcrAuth struct {
	acr string

	log                  *logrus.Entry
	env                  env.Core
	tokenCredential      azcore.TokenCredential
	authenticationClient azcontainerregistry.AuthenticationClient
	now                  func() time.Time

	auth     *types.DockerAuthConfig
	rotateAt time.Time
	mu       sync.RWMutex
}

func NewAcrAuth(acr string, env env.Core, tokenCredential azcore.TokenCredential, authenticationClient azcontainerregistry.AuthenticationClient) *AcrAuth {
	return &AcrAuth{
		acr:                  acr,
		log:                  env.LoggerForComponent("acrAuth"),
		env:                  env,
		tokenCredential:      tokenCredential,
		authenticationClient: authenticationClient,
		now:                  func() time.Time { return time.Now() },
	}
}

func (a *AcrAuth) Get(ctx context.Context) (*types.DockerAuthConfig, error) {
	a.mu.RLock()
	auth, rotateAt := a.auth, a.rotateAt
	a.mu.RUnlock()

	if auth == nil || a.now().After(rotateAt) {
		return a.getNew(ctx)
	}
	return auth, nil
}

func (a *AcrAuth) getNew(ctx context.Context) (*types.DockerAuthConfig, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.log.Info("Retrieving new ACR token")

	getTokenOptions := azcorepolicy.TokenRequestOptions{
		Scopes: []string{a.env.Environment().Cloud.Services[sdkcontainerregistry.ServiceName].Audience + "/.default"},
	}
	aadAccessToken, err := a.tokenCredential.GetToken(ctx, getTokenOptions)
	if err != nil {
		a.log.Errorf("Failed to get AAD access token: %v", err)
		return nil, err
	}

	exchangeAADAccessTokenForACRRefreshTokenOptions := &sdkcontainerregistry.AuthenticationClientExchangeAADAccessTokenForACRRefreshTokenOptions{
		AccessToken: &aadAccessToken.Token,
	}
	acrRefreshTokenResponse, err := a.authenticationClient.ExchangeAADAccessTokenForACRRefreshToken(
		ctx, sdkcontainerregistry.PostContentSchemaGrantTypeAccessToken, a.acr, exchangeAADAccessTokenForACRRefreshTokenOptions,
	)
	if err != nil {
		a.log.Errorf("Failed to get ACR refresh token: %v", err)
		return nil, err
	}

	a.auth = &types.DockerAuthConfig{
		Username: "00000000-0000-0000-0000-000000000000",
		Password: *acrRefreshTokenResponse.RefreshToken,
	}
	a.rotateAt = a.now().Add(rotateAfter)
	return a.auth, nil
}
