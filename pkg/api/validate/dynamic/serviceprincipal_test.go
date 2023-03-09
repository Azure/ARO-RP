package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_aad "github.com/Azure/ARO-RP/pkg/util/mocks/aad"
)

type tokenRequirements struct {
	clientID      string
	clientSecret  string
	tenantID      string
	aadEndpoint   string
	graphEndpoint string
	resource      string
	claims        string
	signMethod    string
}

func TestValidateServicePrincipal(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		tr          *tokenRequirements
		name        string
		wantErr     string
		aadMock     func(*mock_aad.MockTokenClient, *tokenRequirements)
		getTokenErr error
	}{
		{
			name: "pass: Successful Validation",
			tr:   newTokenRequirements(),
		},
		{
			name: "fail: Provided service must not have Application.ReadWrite.OwnedBy permission",
			tr: &tokenRequirements{
				clientID:      "my-client",
				clientSecret:  "my-secret",
				tenantID:      "my-tenant.example.com",
				aadEndpoint:   "https://login.microsoftonline.com/",
				graphEndpoint: "https://graph.windows.net/",
				resource:      "https://management.azure.com/",
				claims:        `{ "Roles":["Application.ReadWrite.OwnedBy"] }`,
			},
			wantErr: "400: InvalidServicePrincipalCredentials: properties.servicePrincipalProfile: The provided service principal must not have the Application.ReadWrite.OwnedBy permission.",
		},
		{
			name:        "fail: Provided service must not have Application.ReadWrite.OwnedBy permission",
			tr:          newTokenRequirements(),
			getTokenErr: fmt.Errorf("parameter activeDirectoryEndpoint cannot be empty"),
			wantErr:     "parameter activeDirectoryEndpoint cannot be empty",
		},
		{
			name: "fail: unavailable signing method",
			tr: &tokenRequirements{
				clientID:      "my-client",
				clientSecret:  "my-secret",
				tenantID:      "my-tenant.example.com",
				aadEndpoint:   "https://login.microsoftonline.com/",
				graphEndpoint: "https://graph.windows.net/",
				resource:      "https://management.azure.com/",
				claims:        `{ "Roles":["Application.ReadWrite.OwnedBy"] }`,
				signMethod:    "fake-signing-method",
			},
			wantErr: "signing method (alg) is unavailable.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			aad := mock_aad.NewMockTokenClient(controller)

			token, err := createToken(tt.tr)
			if err != nil {
				t.Errorf("failed to create testing service principal token: %v\n", err)
			}

			spDynamic, err := NewServicePrincipalValidator(log, &azureclient.PublicCloud, AuthorizerClusterServicePrincipal, aad)
			if err != nil {
				t.Errorf("failed to create ServicePrincipalDynamicValidator: %v\n", err)
			}

			aad.EXPECT().GetToken(ctx, log, tt.tr.clientID, tt.tr.clientSecret, tt.tr.tenantID, tt.tr.aadEndpoint, tt.tr.graphEndpoint).MaxTimes(1).Return(token, tt.getTokenErr)

			err = spDynamic.ValidateServicePrincipal(ctx, tt.tr.clientID, tt.tr.clientSecret, tt.tr.tenantID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%s\n !=\n%s", err, tt.wantErr)
			}
		})
	}
}

// createToken manually creates an adal.ServicePrincipalToken
func createToken(tr *tokenRequirements) (*adal.ServicePrincipalToken, error) {
	if tr.signMethod == "" {
		tr.signMethod = "HS256"
	}
	claimsEnc := base64.StdEncoding.EncodeToString([]byte(tr.claims))
	headerEnc := base64.StdEncoding.EncodeToString([]byte(`{ "alg": "` + tr.signMethod + `", "typ": "JWT" }`))
	signatureEnc := base64.StdEncoding.EncodeToString(
		hmac.New(sha512.New, []byte(headerEnc+claimsEnc+tr.clientSecret)).Sum(nil),
	)

	r := rand.New(rand.NewSource(time.Now().UnixMicro()))
	tk := adal.Token{
		AccessToken:  headerEnc + "." + claimsEnc + "." + signatureEnc,
		RefreshToken: fmt.Sprintf("rand-%d", r.Int()),
		ExpiresIn:    json.Number("300"),
		Resource:     tr.resource,
		Type:         "refresh",
	}

	aadUrl, err := url.Parse(tr.aadEndpoint)
	if err != nil {
		return nil, err
	}
	authUrl, err := url.Parse("https://login.microsoftonline.com/my-tenant.example.com/oauth2/authorize")
	if err != nil {
		return nil, err
	}
	tokenUrl, err := url.Parse("https://login.microsoftonline.com/my-tenant.example.com/oauth2/token")
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	deviceCodeUrl, err := url.Parse("https://devicecode.com")
	if err != nil {
		return nil, err
	}
	return adal.NewServicePrincipalTokenFromManualToken(adal.OAuthConfig{
		AuthorityEndpoint:  *aadUrl,
		AuthorizeEndpoint:  *authUrl,
		TokenEndpoint:      *tokenUrl,
		DeviceCodeEndpoint: *deviceCodeUrl,
	}, tr.clientID, tr.resource, tk)
}

func newTokenRequirements() *tokenRequirements {
	return &tokenRequirements{
		clientID:      "my-client",
		clientSecret:  "my-secret",
		tenantID:      "my-tenant.example.com",
		aadEndpoint:   "https://login.microsoftonline.com/",
		graphEndpoint: "https://graph.windows.net/",
		resource:      "https://management.azure.com/",
		claims:        `{}`,
		signMethod:    "HS256",
	}
}
