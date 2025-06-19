package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
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
					Namespace:         to.Ptr("Microsoft.Authorization"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("Microsoft.Compute"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("Microsoft.Network"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("Microsoft.Storage"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("otherRegisteredProvider"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("otherNotRegisteredProvider"),
					RegistrationState: to.Ptr("NotRegistered"),
				},
			},
		},
		{
			name: "fail: compute not registered",
			mockProviders: []mgmtfeatures.Provider{
				{
					Namespace:         to.Ptr("Microsoft.Authorization"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("Microsoft.Compute"),
					RegistrationState: to.Ptr("NotRegistered"),
				},
				{
					Namespace:         to.Ptr("Microsoft.Network"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("Microsoft.Storage"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("otherRegisteredProvider"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("otherNotRegisteredProvider"),
					RegistrationState: to.Ptr("NotRegistered"),
				},
			},
			wantErr: "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Compute' is not registered.",
		},
		{
			name: "fail: storage missing",
			mockProviders: []mgmtfeatures.Provider{
				{
					Namespace:         to.Ptr("Microsoft.Authorization"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("Microsoft.Compute"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("Microsoft.Network"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("otherRegisteredProvider"),
					RegistrationState: to.Ptr("Registered"),
				},
				{
					Namespace:         to.Ptr("otherNotRegisteredProvider"),
					RegistrationState: to.Ptr("NotRegistered"),
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
