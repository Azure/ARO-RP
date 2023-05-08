package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type tokenRequirements struct {
	clientSecret  string
	claims        jwt.MapClaims
	signingMethod jwt.SigningMethod
}

type signingMethodFake struct{}

func (m signingMethodFake) Verify(signingString, signature string, key interface{}) error {
	return nil
}

func (m signingMethodFake) Sign(signingString string, key interface{}) (string, error) {
	return "", nil
}

func (m signingMethodFake) Alg() string {
	return "fake"
}

func TestValidateServicePrincipal(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		tr      *tokenRequirements
		name    string
		wantErr string
	}{
		{
			name: "pass: Successful Validation",
			tr: &tokenRequirements{
				clientSecret:  "my-secret",
				signingMethod: jwt.SigningMethodHS256,
			},
		},
		{
			name: "fail: Provided service must not have Application.ReadWrite.OwnedBy permission",
			tr: &tokenRequirements{
				clientSecret:  "my-secret",
				claims:        jwt.MapClaims{"roles": []string{"Application.ReadWrite.OwnedBy"}},
				signingMethod: jwt.SigningMethodHS256,
			},
			wantErr: "400: InvalidServicePrincipalCredentials: properties.servicePrincipalProfile: The provided service principal must not have the Application.ReadWrite.OwnedBy permission.",
		},
		{
			name: "fail: unavailable signing method",
			tr: &tokenRequirements{
				clientSecret:  "my-secret",
				claims:        jwt.MapClaims{"roles": []string{"Application.ReadWrite.OwnedBy"}},
				signingMethod: signingMethodFake{},
			},
			wantErr: "400: InvalidServicePrincipalCredentials: properties.servicePrincipalProfile: signing method (alg) is unavailable.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spDynamic := NewServicePrincipalValidator(log, &azureclient.PublicCloud, AuthorizerClusterServicePrincipal)

			err := spDynamic.ValidateServicePrincipal(ctx, tt.tr)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

// GetToken allows tokenRequirements to be used as an azcore.TokenCredential.
func (tr *tokenRequirements) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	token, err := jwt.NewWithClaims(tr.signingMethod, tr.claims).SignedString([]byte(tr.clientSecret))
	if err != nil {
		return azcore.AccessToken{}, err
	}

	return azcore.AccessToken{
		Token:     token,
		ExpiresOn: time.Now().Add(10 * time.Minute),
	}, nil
}
