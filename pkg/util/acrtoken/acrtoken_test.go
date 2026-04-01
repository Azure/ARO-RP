package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_containerregistry "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerregistry"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
)

const (
	tokenName          = "token-12345"
	registryResourceID = "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/arointsvc"
	registryDomain     = "arointsvc.example.com"
)

func TestEnsureTokenAndPassword(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().ACRResourceID().AnyTimes().Return(registryResourceID)

	tokens := mock_containerregistry.NewMockTokensClient(controller)
	tokens.EXPECT().
		CreateAndWait(ctx, "global", "arointsvc", gomock.Any(), mgmtcontainerregistry.Token{
			TokenProperties: &mgmtcontainerregistry.TokenProperties{
				ScopeMapID: pointerutils.ToPtr(registryResourceID + "/scopeMaps/_repositories_pull"),
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
					Value: pointerutils.ToPtr("foo"),
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
	fiftyDaysInThePast := time.Now().UTC().AddDate(0, 0, -50)
	password, err := m.EnsureTokenAndPassword(ctx, &api.RegistryProfile{Username: tokenName, IssueDate: &fiftyDaysInThePast})
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
	env.EXPECT().ACRDomain().AnyTimes().Return(registryDomain)
	r, _ := azure.ParseResourceID(registryResourceID)
	u := deterministicuuid.NewTestUUIDGenerator(0x22)
	now := func() time.Time { return time.UnixMilli(1000) }
	return &manager{
		env:        env,
		r:          r,
		tokens:     tc,
		registries: rc,
		uuid:       u,
		now:        now,
	}
}

func fakeCredentialResult() mgmtcontainerregistry.GenerateCredentialsResult {
	return mgmtcontainerregistry.GenerateCredentialsResult{
		Passwords: &[]mgmtcontainerregistry.TokenPassword{
			{
				Name:  mgmtcontainerregistry.TokenPasswordNamePassword1,
				Value: pointerutils.ToPtr("foo"),
			},
			{
				Name:  mgmtcontainerregistry.TokenPasswordNamePassword2,
				Value: pointerutils.ToPtr("bar"),
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
		TokenID: pointerutils.ToPtr(registryResourceID + "/tokens/" + tokenName),
		Name:    tpn,
	}
}

func TestGetRegistryProfiles(t *testing.T) {
	a := assert.New(t)
	controller := gomock.NewController(t)
	mgr := setupManager(controller, nil, nil)

	ocWithProfile := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			RegistryProfiles: []*api.RegistryProfile{
				{
					Name:     "notwanted.example.com",
					Username: "other",
				},
				{
					Name:     "arointsvc.example.com",
					Username: "foo",
				},
			},
		},
	}
	ocWithoutProfile := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			RegistryProfiles: []*api.RegistryProfile{
				{
					Name:     "notwanted.example.com",
					Username: "other",
				},
			},
		},
	}

	// GetRegistryProfile finds it successfully
	r := mgr.GetRegistryProfile(ocWithProfile)
	a.NotNil(r)
	a.Equal("arointsvc.example.com", r.Name)
	a.Equal("foo", r.Username)

	// GetRegistryProfile can't find it as it doesn't exist
	r = mgr.GetRegistryProfile(ocWithoutProfile)
	a.Nil(r)

	// GetRegistryProfileFromSlice finds it successfully
	r = GetRegistryProfileFromSlice(mgr.env, ocWithProfile.Properties.RegistryProfiles)
	a.NotNil(r)
	a.Equal("arointsvc.example.com", r.Name)
	a.Equal("foo", r.Username)

	// GetRegistryProfileFromSlice can't find it as it doesn't exist
	r = GetRegistryProfileFromSlice(mgr.env, ocWithoutProfile.Properties.RegistryProfiles)
	a.Nil(r)
}

func TestNewAndPutRegistryProfile(t *testing.T) {
	a := assert.New(t)
	controller := gomock.NewController(t)
	mgr := setupManager(controller, nil, nil)

	newProfile := mgr.NewRegistryProfile()
	a.NotNil(newProfile)
	a.Equal("token-22222222-2222-2222-2222-222222220001", newProfile.Username)
	a.Equal("1970-01-01T00:00:01Z", newProfile.IssueDate.Format(time.RFC3339))

	ocWithProfile := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			RegistryProfiles: []*api.RegistryProfile{
				{
					Name:     "arointsvc.example.com",
					Username: "foo",
				},
				{
					Name:     "notwanted.example.com",
					Username: "other",
				},
			},
		},
	}
	ocWithoutProfile := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			RegistryProfiles: []*api.RegistryProfile{
				{
					Name:     "notwanted.example.com",
					Username: "other",
				},
			},
		},
	}

	// If it doesn't exist, it appends it
	mgr.PutRegistryProfile(ocWithoutProfile, newProfile)
	a.Len(ocWithoutProfile.Properties.RegistryProfiles, 2)

	// If it does exist, it replaces it
	mgr.PutRegistryProfile(ocWithProfile, newProfile)
	a.Len(ocWithProfile.Properties.RegistryProfiles, 2)

	// Check that it has been replaced
	aLongTimeAgo := time.UnixMilli(1000)

	for _, err := range deep.Equal(
		ocWithProfile.Properties.RegistryProfiles,
		[]*api.RegistryProfile{
			{
				Name:      "arointsvc.example.com",
				Username:  "token-22222222-2222-2222-2222-222222220001",
				IssueDate: &aLongTimeAgo,
			},
			{
				Name:     "notwanted.example.com",
				Username: "other",
			},
		}) {
		t.Error(err)
	}
}
