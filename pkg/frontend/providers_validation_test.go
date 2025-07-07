package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestValidateProviders(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name            string
		mockProviders   []mgmtfeatures.Provider
		mockProviderErr error
		wantErr         string
	}{
		{
			name: "pass",
			mockProviders: []mgmtfeatures.Provider{
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Authorization"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Compute"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Network"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Storage"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("otherRegisteredProvider"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("otherNotRegisteredProvider"),
					RegistrationState: pointerutils.ToPtr("NotRegistered"),
				},
			},
		},
		{
			name: "fail: compute not registered",
			mockProviders: []mgmtfeatures.Provider{
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Authorization"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Compute"),
					RegistrationState: pointerutils.ToPtr("NotRegistered"),
				},
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Network"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Storage"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("otherRegisteredProvider"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("otherNotRegisteredProvider"),
					RegistrationState: pointerutils.ToPtr("NotRegistered"),
				},
			},
			wantErr: "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Compute' is not registered.",
		},
		{
			name: "fail: storage missing",
			mockProviders: []mgmtfeatures.Provider{
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Authorization"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Compute"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("Microsoft.Network"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("otherRegisteredProvider"),
					RegistrationState: pointerutils.ToPtr("Registered"),
				},
				{
					Namespace:         pointerutils.ToPtr("otherNotRegisteredProvider"),
					RegistrationState: pointerutils.ToPtr("NotRegistered"),
				},
			},
			wantErr: "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Storage' is not registered.",
		},
		{
			name:            "error case",
			mockProviderErr: errors.New("random error"),
			wantErr:         "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			providersClient := mock_features.NewMockProvidersClient(controller)

			providersClient.EXPECT().
				List(gomock.Any(), nil, "").
				Return(tt.mockProviders, tt.mockProviderErr)

			err := validateProviders(ctx, providersClient)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
