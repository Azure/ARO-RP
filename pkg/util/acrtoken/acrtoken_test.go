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

	sdkarmcontainerregistry "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armcontainerregistry "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcontainerregistry"
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

	tokens := mock_armcontainerregistry.NewMockTokensClient(controller)
	tokens.EXPECT().
		CreateAndWait(ctx, "global", "arointsvc", gomock.Any(), sdkarmcontainerregistry.Token{
			Properties: &sdkarmcontainerregistry.TokenProperties{
				ScopeMapID: pointerutils.ToPtr(registryResourceID + "/scopeMaps/_repositories_pull"),
				Status:     pointerutils.ToPtr(sdkarmcontainerregistry.TokenStatusEnabled),
			},
		}).
		Return(nil, nil)

	registries := mock_armcontainerregistry.NewMockRegistriesClient(controller)
	registries.EXPECT().
		GenerateCredentialsAndWait(ctx, "global", "arointsvc", gomock.Any()).
		Return(&sdkarmcontainerregistry.GenerateCredentialsResult{
			Passwords: []*sdkarmcontainerregistry.TokenPassword{
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
		currentTokenPasswords []*sdkarmcontainerregistry.TokenPassword
		wantRenewalName       sdkarmcontainerregistry.TokenPasswordName
		wantPassword          string
	}{
		{
			name:                  "uses password1 when token has no passwords present",
			currentTokenPasswords: []*sdkarmcontainerregistry.TokenPassword{},
			wantRenewalName:       sdkarmcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:          "foo",
		},
		{
			name: "uses password1 when only password2 exists",
			currentTokenPasswords: []*sdkarmcontainerregistry.TokenPassword{
				{
					Name:         pointerutils.ToPtr(sdkarmcontainerregistry.TokenPasswordNamePassword2),
					CreationTime: pointerutils.ToPtr(time.Now()),
				},
			},
			wantRenewalName: sdkarmcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:    "foo",
		},
		{
			name: "uses password2 when only password1 exists",
			currentTokenPasswords: []*sdkarmcontainerregistry.TokenPassword{
				{
					Name:         pointerutils.ToPtr(sdkarmcontainerregistry.TokenPasswordNamePassword1),
					CreationTime: pointerutils.ToPtr(time.Now()),
				},
			},
			wantRenewalName: sdkarmcontainerregistry.TokenPasswordNamePassword2,
			wantPassword:    "bar",
		},
		{
			name: "renews password1 when it is the oldest password",
			currentTokenPasswords: []*sdkarmcontainerregistry.TokenPassword{
				{
					Name:         pointerutils.ToPtr(sdkarmcontainerregistry.TokenPasswordNamePassword1),
					CreationTime: pointerutils.ToPtr(time.Now().Add(-60 * time.Hour * 24)),
				},
				{
					Name:         pointerutils.ToPtr(sdkarmcontainerregistry.TokenPasswordNamePassword2),
					CreationTime: pointerutils.ToPtr(time.Now()),
				},
			},
			wantRenewalName: sdkarmcontainerregistry.TokenPasswordNamePassword1,
			wantPassword:    "foo",
		},
		{
			name: "renews password2 when it is the oldest password",
			currentTokenPasswords: []*sdkarmcontainerregistry.TokenPassword{
				{
					Name:         pointerutils.ToPtr(sdkarmcontainerregistry.TokenPasswordNamePassword1),
					CreationTime: pointerutils.ToPtr(time.Now()),
				},
				{
					Name:         pointerutils.ToPtr(sdkarmcontainerregistry.TokenPasswordNamePassword2),
					CreationTime: pointerutils.ToPtr(time.Now().Add(-60 * time.Hour * 24)),
				},
			},
			wantRenewalName: sdkarmcontainerregistry.TokenPasswordNamePassword2,
			wantPassword:    "bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			tokens := mock_armcontainerregistry.NewMockTokensClient(controller)
			registries := mock_armcontainerregistry.NewMockRegistriesClient(controller)

			tokens.EXPECT().GetTokenProperties(ctx, "global", "arointsvc", tokenName).Return(fakeTokenProperties(tt.currentTokenPasswords), nil)

			registries.EXPECT().GenerateCredentialsAndWait(ctx, "global", "arointsvc", generateCredentialsParameters(tt.wantRenewalName)).Return(fakeCredentialResult(), nil)

			m := setupManager(controller, tokens, registries)

			registryProfile := api.RegistryProfile{
				Username: tokenName,
			}

			err := m.RotateTokenPassword(ctx, &registryProfile)
			if err != nil {
				t.Fatal(err)
			}
			if registryProfile.Password != api.SecureString(tt.wantPassword) {
				t.Errorf("got '%s', want '%s'", registryProfile.Password, tt.wantPassword)
			}
		})
	}
}

func setupManager(controller *gomock.Controller, tc *mock_armcontainerregistry.MockTokensClient, rc *mock_armcontainerregistry.MockRegistriesClient) *manager {
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

func fakeCredentialResult() *sdkarmcontainerregistry.GenerateCredentialsResult {
	return &sdkarmcontainerregistry.GenerateCredentialsResult{
		Passwords: []*sdkarmcontainerregistry.TokenPassword{
			{
				Name:  pointerutils.ToPtr(sdkarmcontainerregistry.TokenPasswordNamePassword1),
				Value: pointerutils.ToPtr("foo"),
			},
			{
				Name:  pointerutils.ToPtr(sdkarmcontainerregistry.TokenPasswordNamePassword2),
				Value: pointerutils.ToPtr("bar"),
			},
		},
	}
}

func fakeTokenProperties(tp []*sdkarmcontainerregistry.TokenPassword) *sdkarmcontainerregistry.TokenProperties {
	return &sdkarmcontainerregistry.TokenProperties{
		Credentials: &sdkarmcontainerregistry.TokenCredentialsProperties{
			Passwords: tp,
		},
	}
}

func generateCredentialsParameters(tpn sdkarmcontainerregistry.TokenPasswordName) sdkarmcontainerregistry.GenerateCredentialsParameters {
	return sdkarmcontainerregistry.GenerateCredentialsParameters{
		TokenID: pointerutils.ToPtr(registryResourceID + "/tokens/" + tokenName),
		Name:    pointerutils.ToPtr(tpn),
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
	r = mgr.GetRegistryProfileFromSlice(ocWithProfile.Properties.RegistryProfiles)
	a.NotNil(r)
	a.Equal("arointsvc.example.com", r.Name)
	a.Equal("foo", r.Username)

	// GetRegistryProfileFromSlice can't find it as it doesn't exist
	r = mgr.GetRegistryProfileFromSlice(ocWithoutProfile.Properties.RegistryProfiles)
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
