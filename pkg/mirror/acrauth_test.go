package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	sdkcontainerregistry "github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_azcontainerregistry "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcontainerregistry"
	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestAcrAuthGet(t *testing.T) {
	ctx := context.Background()

	acr := "arotestsvc.azurecr.io"

	validAadTokenResponse := azcore.AccessToken{
		Token: "token",
	}

	validAcrRefreshTokenResponse := sdkcontainerregistry.AuthenticationClientExchangeAADAccessTokenForACRRefreshTokenResponse{
		ACRRefreshToken: sdkcontainerregistry.ACRRefreshToken{
			RefreshToken: to.Ptr("password"),
		},
	}

	validAcrAuthConfig := &types.DockerAuthConfig{
		Username: "00000000-0000-0000-0000-000000000000",
		Password: "password",
	}

	t.Run("all calls succeed as expected", func(t *testing.T) {
		controller := gomock.NewController(t)
		env := mock_env.NewMockInterface(controller)
		env.EXPECT().
			Environment().
			AnyTimes().
			Return(&azureclient.PublicCloud)

		tokenCredential := mock_azcore.NewMockTokenCredential(controller)
		tokenCredential.EXPECT().
			GetToken(gomock.Any(), gomock.Any()).
			Return(validAadTokenResponse, nil)

		authenticationClient := mock_azcontainerregistry.NewMockAuthenticationClient(controller)
		authenticationClient.EXPECT().
			ExchangeAADAccessTokenForACRRefreshToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(validAcrRefreshTokenResponse, nil)

		acrauth := &AcrAuth{
			acr:                  acr,
			log:                  logrus.NewEntry(logrus.StandardLogger()),
			env:                  env,
			tokenCredential:      tokenCredential,
			authenticationClient: authenticationClient,
			now:                  func() time.Time { return time.Now() },
		}

		acrAuthConfig, err := acrauth.Get(ctx)
		utilerror.AssertErrorMessage(t, err, "")

		if !reflect.DeepEqual(acrAuthConfig, validAcrAuthConfig) {
			t.Errorf("got auth %v, want %v", acrAuthConfig, validAcrAuthConfig)
		}
	})

	t.Run("GetToken returns err", func(t *testing.T) {
		controller := gomock.NewController(t)
		env := mock_env.NewMockInterface(controller)
		env.EXPECT().
			Environment().
			AnyTimes().
			Return(&azureclient.PublicCloud)

		tokenCredential := mock_azcore.NewMockTokenCredential(controller)
		tokenCredential.EXPECT().
			GetToken(gomock.Any(), gomock.Any()).
			Return(azcore.AccessToken{}, fmt.Errorf("unexpected error"))

		acrauth := &AcrAuth{
			acr:             acr,
			log:             logrus.NewEntry(logrus.StandardLogger()),
			env:             env,
			tokenCredential: tokenCredential,
			now:             func() time.Time { return time.Now() },
		}

		_, err := acrauth.Get(ctx)
		utilerror.AssertErrorMessage(t, err, "unexpected error")
	})

	t.Run("ExchangeAADAccessTokenForACRRefreshToken returns err", func(t *testing.T) {
		controller := gomock.NewController(t)
		env := mock_env.NewMockInterface(controller)
		env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)

		tokenCredential := mock_azcore.NewMockTokenCredential(controller)
		tokenCredential.EXPECT().
			GetToken(gomock.Any(), gomock.Any()).
			Return(validAadTokenResponse, nil)

		authenticationClient := mock_azcontainerregistry.NewMockAuthenticationClient(controller)
		authenticationClient.EXPECT().
			ExchangeAADAccessTokenForACRRefreshToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(sdkcontainerregistry.AuthenticationClientExchangeAADAccessTokenForACRRefreshTokenResponse{}, fmt.Errorf("unexpected error"))

		acrauth := &AcrAuth{
			acr:                  acr,
			log:                  logrus.NewEntry(logrus.StandardLogger()),
			env:                  env,
			tokenCredential:      tokenCredential,
			authenticationClient: authenticationClient,
			now:                  func() time.Time { return time.Now() },
		}

		_, err := acrauth.Get(ctx)
		utilerror.AssertErrorMessage(t, err, "unexpected error")
	})

	t.Run("subsequent calls use cached token", func(t *testing.T) {
		controller := gomock.NewController(t)
		env := mock_env.NewMockInterface(controller)
		env.EXPECT().
			Environment().
			AnyTimes().
			Return(&azureclient.PublicCloud)

		tokenCredential := mock_azcore.NewMockTokenCredential(controller)
		tokenCredential.EXPECT().
			GetToken(gomock.Any(), gomock.Any()).
			Times(1).
			Return(validAadTokenResponse, nil)

		authenticationClient := mock_azcontainerregistry.NewMockAuthenticationClient(controller)
		authenticationClient.EXPECT().
			ExchangeAADAccessTokenForACRRefreshToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1).
			Return(validAcrRefreshTokenResponse, nil)

		acrauth := &AcrAuth{
			acr:                  acr,
			log:                  logrus.NewEntry(logrus.StandardLogger()),
			env:                  env,
			tokenCredential:      tokenCredential,
			authenticationClient: authenticationClient,
			now:                  func() time.Time { return time.Now() },
		}

		for range 10 {
			_, err := acrauth.Get(ctx)
			utilerror.AssertErrorMessage(t, err, "")
		}
	})

	t.Run("subsequent calls after token expiration time request new token", func(t *testing.T) {
		controller := gomock.NewController(t)
		env := mock_env.NewMockInterface(controller)
		env.EXPECT().
			Environment().
			AnyTimes().
			Return(&azureclient.PublicCloud)

		tokenCredential := mock_azcore.NewMockTokenCredential(controller)
		tokenCredential.EXPECT().
			GetToken(gomock.Any(), gomock.Any()).
			Times(2).
			Return(validAadTokenResponse, nil)

		authenticationClient := mock_azcontainerregistry.NewMockAuthenticationClient(controller)
		authenticationClient.EXPECT().
			ExchangeAADAccessTokenForACRRefreshToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(2).
			Return(validAcrRefreshTokenResponse, nil)

		mockNow := time.Now()

		acrauth := &AcrAuth{
			acr:                  acr,
			log:                  logrus.NewEntry(logrus.StandardLogger()),
			env:                  env,
			tokenCredential:      tokenCredential,
			authenticationClient: authenticationClient,
			now:                  func() time.Time { return mockNow },
		}

		_, err := acrauth.Get(ctx)
		utilerror.AssertErrorMessage(t, err, "")

		mockNow = mockNow.Add(3 * time.Hour)

		_, err = acrauth.Get(ctx)
		utilerror.AssertErrorMessage(t, err, "")
	})

	t.Run("concurrent calls will sync and wait", func(t *testing.T) {
		controller := gomock.NewController(t)
		env := mock_env.NewMockInterface(controller)
		env.EXPECT().
			Environment().
			AnyTimes().
			Return(&azureclient.PublicCloud)

		tokenCredential := mock_azcore.NewMockTokenCredential(controller)
		tokenCredential.EXPECT().
			GetToken(gomock.Any(), gomock.Any()).
			Times(1).
			Return(validAadTokenResponse, nil)

		authenticationClient := mock_azcontainerregistry.NewMockAuthenticationClient(controller)
		authenticationClient.EXPECT().
			ExchangeAADAccessTokenForACRRefreshToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1).
			Return(validAcrRefreshTokenResponse, nil)

		acrauth := &AcrAuth{
			acr:                  acr,
			log:                  logrus.NewEntry(logrus.StandardLogger()),
			env:                  env,
			tokenCredential:      tokenCredential,
			authenticationClient: authenticationClient,
			now:                  func() time.Time { return time.Now() },
		}

		var wg sync.WaitGroup

		for range 10 {
			wg.Add(1)
			go func() { acrauth.Get(ctx); wg.Done() }()
		}
		wg.Wait()
	})
}
