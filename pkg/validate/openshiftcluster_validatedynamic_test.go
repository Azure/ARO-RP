package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
//
import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type token struct {
	accesstoken string
}

func (t token) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: t.accesstoken, ExpiresOn: time.Now().Add(24 * time.Hour)}, nil
}

func TestEnsureAccessTokenClaims(t *testing.T) {
	// generated, not real tokens
	hasOid := "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJia0Biay5jb20iLCJuYW1lIjoiQmlsbHkgS2VpbGxvciIsICJvaWQiOiAiMTExMTEiLCAiaWF0IjoxNTQ2MzAwODAwLCJleHAiOjE4OTM0NTYwMDB9."
	hasAltsecid := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwgImFsdHNlY2lkIjoiYmlsbHkga2VpbGxvciIsICJuYW1lIjoiSm9obiBEb2UiLCJpYXQiOjE1MTYyMzkwMjJ9."
	hasPuid := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwgInB1aWQiOiJiaWxseSBrZWlsbG9yIiwgIm5hbWUiOiJKb2huIERvZSIsImlhdCI6MTUxNjIzOTAyMn0."
	for _, tt := range []struct {
		name         string
		tokenFactory func() azcore.TokenCredential
		wantErr      string
	}{
		{
			name: "server error",
			tokenFactory: func() azcore.TokenCredential {
				cred := fake.TokenCredential{}
				cred.SetError(errors.New("Unable to establish a connection to ARM"))
				return &cred
			},
			wantErr: "Unable to establish a connection to ARM",
		},
		{
			name: "invalid Token, no required claims at all",
			tokenFactory: func() azcore.TokenCredential {
				return &fake.TokenCredential{}
			},
			wantErr: "400: InvalidServicePrincipalToken: properties.servicePrincipalProfile: The provided service principal generated an invalid token.",
		},
		{
			name: "valid token: has oid",
			tokenFactory: func() azcore.TokenCredential {
				return token{hasOid}
			},
		}, {
			name: "valid token: has altsecid",
			tokenFactory: func() azcore.TokenCredential {
				return token{hasAltsecid}
			},
		}, {
			name: "valid token: has puid",
			tokenFactory: func() azcore.TokenCredential {
				return token{hasPuid}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			scopes := []string{}
			tok := tt.tokenFactory()

			err := ensureAccessTokenClaims(ctx, tok, scopes)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
