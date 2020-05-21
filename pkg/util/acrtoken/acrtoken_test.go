package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	mock_containerregistry "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerregistry"
)

func TestEnsureTokenAndPassword(t *testing.T) {
	ctx := context.Background()
	env := &env.Test{}

	controller := gomock.NewController(t)
	defer controller.Finish()

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
