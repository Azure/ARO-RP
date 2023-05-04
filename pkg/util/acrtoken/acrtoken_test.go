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
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_containerregistry "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerregistry"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestEnsureTokenAndPassword(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().ACRResourceID().AnyTimes().Return("/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/arointsvc")

	tokens := mock_containerregistry.NewMockTokensClient(controller)
	tokens.EXPECT().
		CreateAndWait(ctx, "global", "arointsvc", gomock.Any(), mgmtcontainerregistry.Token{
			TokenProperties: &mgmtcontainerregistry.TokenProperties{
				ScopeMapID: to.StringPtr(env.ACRResourceID() + "/scopeMaps/_repositories_pull"),
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

	password, err := m.EnsureTokenAndPassword(ctx, &api.RegistryProfile{Username: "token-12345"})
	if err != nil {
		t.Fatal(err)
	}
	if password != "foo" {
		t.Error(password)
	}
}

func TestRotateTokenPassword(t *testing.T) {
	tests := []struct {
		name                    string
		tokenPasswordProperties []mgmtcontainerregistry.TokenPassword
		wantRenewalName         mgmtcontainerregistry.TokenPasswordName
		wantPassword            string
	}{
		{
			name:                    "uses password1 when token has no passwords present",
			tokenPasswordProperties: []mgmtcontainerregistry.TokenPassword{},
			wantRenewalName:         mgmtcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:            "foo",
		},
		{
			name: "uses password1 when only password2 exists",
			tokenPasswordProperties: []mgmtcontainerregistry.TokenPassword{
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword2,
					CreationTime: &date.Time{Time: time.Now()},
				},
			},
			wantRenewalName: mgmtcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:    "foo",
		},
		{
			name: "uses password2 when only password1 exists",
			tokenPasswordProperties: []mgmtcontainerregistry.TokenPassword{
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword1,
					CreationTime: &date.Time{Time: time.Now()},
				},
			},
			wantRenewalName: mgmtcontainerregistry.TokenPasswordNamePassword2,
			wantPassword:    "bar",
		},
		{
			name: "renews password1 when it is the oldest password",
			tokenPasswordProperties: []mgmtcontainerregistry.TokenPassword{
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword1,
					CreationTime: &date.Time{Time: time.Now().Add(-60 * time.Hour * 24)},
				},
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword2,
					CreationTime: &date.Time{Time: time.Now()},
				},
			},
			wantRenewalName: mgmtcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:    "foo",
		},
		{
			name: "renews password2 when it is the oldest password",
			tokenPasswordProperties: []mgmtcontainerregistry.TokenPassword{
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword1,
					CreationTime: &date.Time{Time: time.Now()},
				},
				{
					Name:         mgmtcontainerregistry.TokenPasswordNamePassword2,
					CreationTime: &date.Time{Time: time.Now().Add(-60 * time.Hour * 24)},
				},
			},
			wantRenewalName: mgmtcontainerregistry.TokenPasswordNamePassword2,
			wantPassword:    "bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username := "aro-12345"
			resourceID := "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/arointsvc"
			ctx := context.Background()
			controller := gomock.NewController(t)

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ACRResourceID().AnyTimes().Return(resourceID)
			tokens := mock_containerregistry.NewMockTokensClient(controller)
			registries := mock_containerregistry.NewMockRegistriesClient(controller)
			r, err := azure.ParseResourceID(env.ACRResourceID())
			if err != nil {
				t.Fatal(err)
			}
			credentialResult := mgmtcontainerregistry.GenerateCredentialsResult{
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
			registryProfile := api.RegistryProfile{
				Username: username,
			}

			fakeTokenProperties := mgmtcontainerregistry.TokenProperties{
				Credentials: &mgmtcontainerregistry.TokenCredentialsProperties{
					Passwords: &tt.tokenPasswordProperties,
				},
			}
			tokens.EXPECT().GetTokenProperties(ctx, "global", "arointsvc", username).Return(fakeTokenProperties, nil)

			expectedCredentialParameters := mgmtcontainerregistry.GenerateCredentialsParameters{
				TokenID: to.StringPtr(resourceID + "/tokens/" + username),
				Name:    tt.wantRenewalName,
			}
			registries.EXPECT().GenerateCredentials(ctx, "global", "arointsvc", expectedCredentialParameters).
				Return(credentialResult, nil)

			m := &manager{
				env: env,
				r:   r,

				registries: registries,
				tokens:     tokens,
			}

			err = m.RotateTokenPassword(ctx, &registryProfile)
			if err != nil {
				t.Fatal(err)
			}
			if registryProfile.Password != api.SecureString(tt.wantPassword) {
				t.Error(registryProfile.Password)
			}
		})
	}
}
