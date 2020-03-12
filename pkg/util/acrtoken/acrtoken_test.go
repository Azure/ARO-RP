package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	azcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	mockcr "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerregistry"
)

func TestManagerCreateToken(t *testing.T) {
	ctx := context.Background()
	e := &env.Test{}
	m := &manager{
		env:      e,
		scopeMap: fmt.Sprintf("%s/scopeMap/_repositories_pull", e.ACRResourceID()),
	}

	gmc := gomock.NewController(t)
	defer gmc.Finish()

	mtc := mockcr.NewMockTokensClient(gmc)
	mtc.EXPECT().CreateAndWait(ctx, "global", "arosvc", gomock.Any(), gomock.Any()).Return(nil)
	m.tokens = mtc

	err := m.CreateToken(ctx, "token-12345")
	if err != nil {
		t.Errorf("manager.CreateToken() error = %v", err)
	}
}

func TestManagerCreatePassword(t *testing.T) {
	ctx := context.Background()
	e := &env.Test{}
	m := &manager{
		env:      e,
		scopeMap: fmt.Sprintf("%s/scopeMap/_repositories_pull", e.ACRResourceID()),
	}

	gmc := gomock.NewController(t)
	defer gmc.Finish()

	mrc := mockcr.NewMockRegistriesClient(gmc)
	gcr := azcontainerregistry.GenerateCredentialsResult{
		Passwords: &[]azcontainerregistry.TokenPassword{
			{
				Value: to.StringPtr("foo"),
			},
		},
	}
	mrc.EXPECT().GenerateCredentials(ctx, "global", "arosvc", gomock.Any()).Return(gcr, nil)
	m.registries = mrc

	oc := api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			RegistryProfiles: []api.RegistryProfile{
				{
					Type:     api.RegistryTypeACR,
					Username: "token-2345",
				},
			},
		},
	}

	pass, err := m.CreatePassword(ctx, &oc)
	if err != nil {
		t.Errorf("manager.CreatePassword() error = %v", err)
	}
	if pass != "foo" {
		t.Errorf("manager.CreatePassword() password = %v", pass)
	}
}

func TestManagerCreatePasswordExisting(t *testing.T) {
	ctx := context.Background()
	e := &env.Test{}
	m := &manager{
		env: e,
	}
	oc := api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			RegistryProfiles: []api.RegistryProfile{
				{
					Type:     api.RegistryTypeACR,
					Username: "token-2345",
					Password: "foo",
				},
			},
		},
	}

	pass, err := m.CreatePassword(ctx, &oc)
	if err != nil {
		t.Errorf("manager.CreatePassword() error = %v", err)
	}
	if pass != "foo" {
		t.Errorf("manager.CreatePassword() password = %v", pass)
	}
}
