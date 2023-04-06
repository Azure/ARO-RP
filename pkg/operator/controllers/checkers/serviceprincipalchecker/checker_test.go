package serviceprincipalchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	mock_dynamic "github.com/Azure/ARO-RP/pkg/util/mocks/dynamic"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestCheck(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())
	mockCredentials := &clusterauthorizer.Credentials{
		ClientID:     []byte("fake-client-id"),
		ClientSecret: []byte("fake-client-secret"),
		TenantID:     []byte("fake-tenant-id"),
	}

	for _, tt := range []struct {
		name             string
		credentialsExist bool
		validator        func(controller *gomock.Controller) dynamic.ServicePrincipalValidator
		wantErr          string
	}{
		{
			name:             "valid service principal",
			credentialsExist: true,
			validator: func(controller *gomock.Controller) dynamic.ServicePrincipalValidator {
				validator := mock_dynamic.NewMockDynamic(controller)
				validator.EXPECT().ValidateServicePrincipal(ctx, string(mockCredentials.ClientID), string(mockCredentials.ClientSecret), string(mockCredentials.TenantID))
				return validator
			},
		},
		{
			name:             "could not instantiate a validator",
			credentialsExist: true,
			validator: func(controller *gomock.Controller) dynamic.ServicePrincipalValidator {
				validator := mock_dynamic.NewMockDynamic(controller)
				validator.EXPECT().ValidateServicePrincipal(ctx, string(mockCredentials.ClientID), string(mockCredentials.ClientSecret), string(mockCredentials.TenantID)).
					Return(errors.New("fake validation error"))
				return validator
			},
			wantErr: "fake validation error",
		},
		{
			name:             "could not instantiate a validator",
			credentialsExist: true,
			wantErr:          "fake validator constructor error",
		},
		{
			name:    "could not get service principal credentials",
			wantErr: "fake credentials get error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			var validatorMock dynamic.ServicePrincipalValidator
			if tt.validator != nil {
				validatorMock = tt.validator(controller)
			}

			sp := &checker{
				log: log,
				credentials: func(ctx context.Context) (*clusterauthorizer.Credentials, error) {
					if tt.credentialsExist {
						return mockCredentials, nil
					}
					return nil, errors.New("fake credentials get error")
				},
				newSPValidator: func(azEnv *azureclient.AROEnvironment) (dynamic.ServicePrincipalValidator, error) {
					if validatorMock != nil {
						return validatorMock, nil
					}
					return nil, errors.New("fake validator constructor error")
				},
			}

			err := sp.Check(ctx, azuretypes.PublicCloud.Name())
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
