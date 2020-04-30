package aad

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	mock_instancemetadata "github.com/Azure/ARO-RP/pkg/util/mocks/instancemetadata"
)

func TestAuthenticateAndGetToken(t *testing.T) {
	ctx := context.Background()

	// Example of test JWTs used below
	// {
	// 	"iss": "TestValidateServicePrincipalProfile",
	// 	"iat": 1588217256,
	// 	"exp": 1619753256,
	// 	"aud": "www.example.com",
	// 	"sub": "test@example.com",
	// 	"altsecid": "ok"
	// }

	missingClaimJWT := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJUZXN0VmFsaWRhdGVTZXJ2aWNlUHJpbmNpcGFsUHJvZmlsZSIsImlhdCI6MTU4ODIxNzI1NiwiZXhwIjoxNjE5NzUzMjU2LCJhdWQiOiJ3d3cuZXhhbXBsZS5jb20iLCJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIn0.bE-3HRgSvzQMK959cpPM_JKFZC-kO-MqdyB3btXJw5U"
	altsecidJWT := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJUZXN0VmFsaWRhdGVTZXJ2aWNlUHJpbmNpcGFsUHJvZmlsZSIsImlhdCI6MTU4ODIxNzI1NiwiZXhwIjoxNjE5NzUzMjU2LCJhdWQiOiJ3d3cuZXhhbXBsZS5jb20iLCJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwiYWx0c2VjaWQiOiJvayJ9.P4ETdlihD2YNGB9b4ARYX7IIEudP4f7a2xHcNCMzER8"
	oidJWT := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJUZXN0VmFsaWRhdGVTZXJ2aWNlUHJpbmNpcGFsUHJvZmlsZSIsImlhdCI6MTU4ODIxNzI1NiwiZXhwIjoxNjE5NzUzMjU2LCJhdWQiOiJ3d3cuZXhhbXBsZS5jb20iLCJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwib2lkIjoib2sifQ.N59nPmaMZo8ZcRgNKG_izLX6GQ99INkoum9fObbF2TY"
	puidJWT := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJUZXN0VmFsaWRhdGVTZXJ2aWNlUHJpbmNpcGFsUHJvZmlsZSIsImlhdCI6MTU4ODIxNzI1NiwiZXhwIjoxNjE5NzUzMjU2LCJhdWQiOiJ3d3cuZXhhbXBsZS5jb20iLCJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwicHVpZCI6Im9rIn0.qhXyekqpUkIfFlDZEo7VPyIEAPM6CZDCUqnZcWnVyiY"

	log := logrus.NewEntry(logrus.StandardLogger())

	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ServicePrincipalProfile: api.ServicePrincipalProfile{
				TenantID:     "1234",
				ClientID:     "5678",
				ClientSecret: api.SecureString("shhh"),
			},
		},
	}

	for _, tt := range []struct {
		name    string
		timeout time.Duration
		mocks   func(token *mock_instancemetadata.MockServicePrincipalToken)
		wantErr string
	}{
		{
			name:    "RefreshWithContext error",
			timeout: 500 * time.Millisecond,
			mocks: func(token *mock_instancemetadata.MockServicePrincipalToken) {
				token.EXPECT().RefreshWithContext(ctx).Return(fmt.Errorf("adal: Failed to build the refresh request"))

			},
			wantErr: "400: InvalidServicePrincipalCredentials: properties.servicePrincipalProfile: The provided service principal credentials are invalid.",
		},
		{
			name:    "Token is missing required claims",
			timeout: 500 * time.Millisecond,
			mocks: func(token *mock_instancemetadata.MockServicePrincipalToken) {
				token.EXPECT().RefreshWithContext(ctx).Return(nil).AnyTimes()
				token.EXPECT().OAuthToken().Return(missingClaimJWT)

			},
			wantErr: "400: InvalidServicePrincipalClaims: properties.servicePrincipalProfile: The provided service principal does not give an access token with at least one of the claims 'altsecid', 'oid' or 'puid'.",
		},
		{
			name:    "AADSTS700016 error (slow AAD propagation), retry, then timeout",
			timeout: 500 * time.Millisecond,
			mocks: func(token *mock_instancemetadata.MockServicePrincipalToken) {
				token.EXPECT().RefreshWithContext(ctx).Return(errors.New("AADSTS700016"))
			},
			wantErr: "400: InvalidServicePrincipalCredentials: properties.servicePrincipalProfile: The provided service principal credentials are invalid.",
		},
		{
			name:    "AADSTS700016 error (slow AAD propagation), retry, then success with altsecid claim",
			timeout: 5 * time.Second,
			mocks: func(token *mock_instancemetadata.MockServicePrincipalToken) {
				token.EXPECT().RefreshWithContext(ctx).Return(errors.New("AADSTS700016"))
				token.EXPECT().RefreshWithContext(ctx).Return(nil)
				token.EXPECT().OAuthToken().Return(altsecidJWT)
			},
		},
		{
			name:    "AADSTS700016 error (slow AAD propagation), retry, then success with oid claim",
			timeout: 5 * time.Second,
			mocks: func(token *mock_instancemetadata.MockServicePrincipalToken) {
				token.EXPECT().RefreshWithContext(ctx).Return(errors.New("AADSTS700016"))
				token.EXPECT().RefreshWithContext(ctx).Return(nil)
				token.EXPECT().OAuthToken().Return(oidJWT)
			},
		},
		{
			name:    "AADSTS700016 error (slow AAD propagation), retry, then success with puid claim",
			timeout: 5 * time.Second,
			mocks: func(token *mock_instancemetadata.MockServicePrincipalToken) {
				token.EXPECT().RefreshWithContext(ctx).Return(errors.New("AADSTS700016"))
				token.EXPECT().RefreshWithContext(ctx).Return(nil)
				token.EXPECT().OAuthToken().Return(puidJWT)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			token := mock_instancemetadata.NewMockServicePrincipalToken(controller)
			tt.mocks(token)

			tf := TokenFactory{
				NewToken: func(conf auth.ClientCredentialsConfig) (instancemetadata.ServicePrincipalToken, error) {
					return token, nil
				},
				RetryInterval: 1 * time.Second,
				Timeout:       tt.timeout,
			}

			_, err := tf.AuthenticateAndGetToken(ctx, log, oc, "test")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
