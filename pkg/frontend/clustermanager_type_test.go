package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_api "github.com/Azure/ARO-RP/pkg/util/mocks/api"
)

func TestValidateOcmResourceType(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	for _, tt := range []struct {
		name                          string
		vars                          map[string]string
		apiVersion                    string
		syncSetConverter              api.SyncSetConverter
		machinePoolConverter          api.MachinePoolConverter
		syncIdentityProviderConverter api.SyncIdentityProviderConverter
		secretConverter               api.SecretConverter
		err                           string
	}{
		{
			name: "syncset - resource type is valid",
			vars: map[string]string{
				"ocmResourceType":           "syncset",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			syncSetConverter: mock_api.NewMockSyncSetConverter(controller),
			err:              "",
		},
		{
			name: "syncset - resource type is invalid",
			vars: map[string]string{
				"ocmResourceType":           "invalid",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			syncSetConverter: mock_api.NewMockSyncSetConverter(controller),
			err:              "the resource type 'invalid' is not valid for api version '2022-09-04'",
		},
		{
			name: "syncset - converter is nil",
			vars: map[string]string{
				"ocmResourceType":           "syncset",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			syncSetConverter: nil,
			err:              "the resource type 'syncset' is not valid for api version '2022-09-04'",
		},
		{
			name: "machinepool - resource type is valid",
			vars: map[string]string{
				"ocmResourceType":           "machinepool",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			machinePoolConverter: mock_api.NewMockMachinePoolConverter(controller),
			err:                  "",
		},
		{
			name: "machinepool - resource type is invalid",
			vars: map[string]string{
				"ocmResourceType":           "invalid",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			machinePoolConverter: mock_api.NewMockMachinePoolConverter(controller),
			err:                  "the resource type 'invalid' is not valid for api version '2022-09-04'",
		},
		{
			name: "machinepool - converter is nil",
			vars: map[string]string{
				"ocmResourceType":           "machinepool",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			machinePoolConverter: nil,
			err:                  "the resource type 'machinepool' is not valid for api version '2022-09-04'",
		},
		{
			name: "syncidentityprovider - resource type is valid",
			vars: map[string]string{
				"ocmResourceType":           "syncidentityprovider",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			syncIdentityProviderConverter: mock_api.NewMockSyncIdentityProviderConverter(controller),
			err:                           "",
		},
		{
			name: "syncidentityprovider - resource type is invalid",
			vars: map[string]string{
				"ocmResourceType":           "invalid",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			syncIdentityProviderConverter: mock_api.NewMockSyncIdentityProviderConverter(controller),
			err:                           "the resource type 'invalid' is not valid for api version '2022-09-04'",
		},
		{
			name: "syncidentityprovider - converter is nil",
			vars: map[string]string{
				"ocmResourceType":           "syncidentityprovider",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			syncIdentityProviderConverter: nil,
			err:                           "the resource type 'syncidentityprovider' is not valid for api version '2022-09-04'",
		},
		{
			name: "secret - resource type is valid",
			vars: map[string]string{
				"ocmResourceType":           "secret",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			secretConverter: mock_api.NewMockSecretConverter(controller),
			err:             "",
		},
		{
			name: "secret - resource type is invalid",
			vars: map[string]string{
				"ocmResourceType":           "invalid",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			secretConverter: mock_api.NewMockSecretConverter(controller),
			err:             "the resource type 'invalid' is not valid for api version '2022-09-04'",
		},
		{
			name: "secret - converter is nil",
			vars: map[string]string{
				"ocmResourceType":           "secret",
				"api-version":               "2022-09-04",
				"resourceProviderNamespace": "microsoft.redhatopenshift",
			},
			secretConverter: nil,
			err:             "the resource type 'secret' is not valid for api version '2022-09-04'",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &frontend{
				apis: map[string]*api.Version{
					"2022-09-04": {
						SyncSetConverter:              tt.syncSetConverter,
						MachinePoolConverter:          tt.machinePoolConverter,
						SyncIdentityProviderConverter: tt.syncIdentityProviderConverter,
						SecretConverter:               tt.secretConverter,
					},
				},
			}

			err := f.validateOcmResourceType(tt.vars)
			if err != nil {
				if err.Error() != tt.err {
					t.Errorf("wanted '%v', got '%v'", tt.err, err)
				}
			}
		})
	}
}
