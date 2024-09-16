package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_containerregistry "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerregistry"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

const (
	tokenName          = "token-12345"
	registryResourceID = "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/arointsvc"
)

func TestEnsureTokenAndPassword(t *testing.T) {
	ctx := context.Background()
	var tokenExpiration = time.Now().UTC().Add(time.Hour * 24 * 90)

	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().ACRResourceID().AnyTimes().Return(registryResourceID)

	tokens := mock_containerregistry.NewMockTokensClient(controller)
	tokens.EXPECT().
		CreateAndWait(ctx, "global", "arointsvc", gomock.Any(), mgmtcontainerregistry.Token{
			TokenProperties: &mgmtcontainerregistry.TokenProperties{
				ScopeMapID: to.StringPtr(registryResourceID + "/scopeMaps/_repositories_pull"),
				Status:     mgmtcontainerregistry.TokenStatusEnabled,
			},
		}).
		Return(nil)

	registries := mock_containerregistry.NewMockRegistriesClient(controller)
	registries.EXPECT().
		GenerateCredentials(ctx, "global", "arointsvc", gomock.Any()).
		Return(mgmtcontainerregistry.GenerateCredentialsResult{
			Passwords: &[]mgmtcontainerregistry.TokenPassword{
				{
					Value: to.StringPtr("foo"),
				},
			},
		}, nil)

	r, err := azure.ParseResourceID(env.ACRResourceID())
	if err != nil {
		t.Fatal(err)
	}

	m := &manager{
		env: env,
		r:   r,

		registries: registries,
		tokens:     tokens,
	}

	password, err := m.EnsureTokenAndPassword(ctx, &api.RegistryProfile{Username: tokenName, IssueDate: &date.Time{Time: tokenExpiration}})
	if err != nil {
		t.Fatal(err)
	}
	if password != "foo" {
		t.Error(password)
	}
}

func TestRotateTokenPassword(t *testing.T) {
	tests := []struct {
		name                  string
		currentTokenPasswords []mgmtcontainerregistry.TokenPassword
		wantRenewalName       mgmtcontainerregistry.TokenPasswordName
		wantPassword          string
	}{
		{
			name:                  "uses password1 when token has no passwords present",
			currentTokenPasswords: []mgmtcontainerregistry.TokenPassword{},
			wantRenewalName:       mgmtcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:          "foo",
		},
		{
			name: "uses password1 when only password2 exists",
			currentTokenPasswords: []mgmtcontainerregistry.TokenPassword{
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword2,
					CreationTime: toDate(time.Now()),
				},
			},
			wantRenewalName: mgmtcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:    "foo",
		},
		{
			name: "uses password2 when only password1 exists",
			currentTokenPasswords: []mgmtcontainerregistry.TokenPassword{
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword1,
					CreationTime: toDate(time.Now()),
				},
			},
			wantRenewalName: mgmtcontainerregistry.TokenPasswordNamePassword2,
			wantPassword:    "bar",
		},
		{
			name: "renews password1 when it is the oldest password",
			currentTokenPasswords: []mgmtcontainerregistry.TokenPassword{
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword1,
					CreationTime: toDate(time.Now().Add(-60 * time.Hour * 24)),
				},
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword2,
					CreationTime: toDate(time.Now()),
				},
			},
			wantRenewalName: mgmtcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:    "foo",
		},
		{
			name: "renews password2 when it is the oldest password",
			currentTokenPasswords: []mgmtcontainerregistry.TokenPassword{
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword1,
					CreationTime: toDate(time.Now()),
				},
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword2,
					CreationTime: toDate(time.Now().Add(-60 * time.Hour * 24)),
				},
			},
			wantRenewalName: mgmtcontainerregistry.TokenPasswordNamePassword2,
			wantPassword:    "bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			tokens := mock_containerregistry.NewMockTokensClient(controller)
			registries := mock_containerregistry.NewMockRegistriesClient(controller)

			tokens.EXPECT().GetTokenProperties(ctx, "global", "arointsvc", tokenName).Return(fakeTokenProperties(&tt.currentTokenPasswords), nil)

			registries.EXPECT().GenerateCredentials(ctx, "global", "arointsvc", generateCredentialsParameters(tt.wantRenewalName)).Return(fakeCredentialResult(), nil)

			m := setupManager(controller, tokens, registries)

			registryProfile := api.RegistryProfile{
				Username: tokenName,
			}

			err := m.RotateTokenPassword(ctx, &registryProfile)
			if err != nil {
				t.Fatal(err)
			}
			if registryProfile.Password != api.SecureString(tt.wantPassword) {
				t.Error(registryProfile.Password)
			}
		})
	}
}

func toDate(t time.Time) *date.Time {
	return &date.Time{Time: t}
}

func setupManager(controller *gomock.Controller, tc *mock_containerregistry.MockTokensClient, rc *mock_containerregistry.MockRegistriesClient) *manager {
	env := mock_env.NewMockInterface(controller)
	env.EXPECT().ACRResourceID().AnyTimes().Return(registryResourceID)
	r, _ := azure.ParseResourceID(registryResourceID)
	return &manager{
		env:        env,
		r:          r,
		tokens:     tc,
		registries: rc,
	}
}

func fakeCredentialResult() mgmtcontainerregistry.GenerateCredentialsResult {
	return mgmtcontainerregistry.GenerateCredentialsResult{
		Passwords: &[]mgmtcontainerregistry.TokenPassword{
			{
				Name:  mgmtcontainerregistry.TokenPasswordNamePassword1,
				Value: to.StringPtr("foo"),
			},
			{
				Name:  mgmtcontainerregistry.TokenPasswordNamePassword2,
				Value: to.StringPtr("bar"),
			},
		},
	}
}

func fakeTokenProperties(tp *[]mgmtcontainerregistry.TokenPassword) mgmtcontainerregistry.TokenProperties {
	return mgmtcontainerregistry.TokenProperties{
		Credentials: &mgmtcontainerregistry.TokenCredentialsProperties{
			Passwords: tp,
		},
	}
}

func generateCredentialsParameters(tpn mgmtcontainerregistry.TokenPasswordName) mgmtcontainerregistry.GenerateCredentialsParameters {
	return mgmtcontainerregistry.GenerateCredentialsParameters{
		TokenID: to.StringPtr(registryResourceID + "/tokens/" + tokenName),
		Name:    tpn,
	}
}
