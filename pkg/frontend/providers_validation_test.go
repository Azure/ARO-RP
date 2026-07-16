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
					Namespace:         new("Microsoft.Authorization"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("Microsoft.Compute"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("Microsoft.Network"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("Microsoft.Storage"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("otherRegisteredProvider"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("otherNotRegisteredProvider"),
					RegistrationState: new("NotRegistered"),
				},
			},
		},
		{
			name: "fail: compute not registered",
			mockProviders: []mgmtfeatures.Provider{
				{
					Namespace:         new("Microsoft.Authorization"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("Microsoft.Compute"),
					RegistrationState: new("NotRegistered"),
				},
				{
					Namespace:         new("Microsoft.Network"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("Microsoft.Storage"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("otherRegisteredProvider"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("otherNotRegisteredProvider"),
					RegistrationState: new("NotRegistered"),
				},
			},
			wantErr: "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Compute' is not registered.",
		},
		{
			name: "fail: storage missing",
			mockProviders: []mgmtfeatures.Provider{
				{
					Namespace:         new("Microsoft.Authorization"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("Microsoft.Compute"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("Microsoft.Network"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("otherRegisteredProvider"),
					RegistrationState: new("Registered"),
				},
				{
					Namespace:         new("otherNotRegisteredProvider"),
					RegistrationState: new("NotRegistered"),
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
