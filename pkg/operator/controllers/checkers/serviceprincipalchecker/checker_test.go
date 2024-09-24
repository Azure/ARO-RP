package serviceprincipalchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest/azure"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_dynamic "github.com/Azure/ARO-RP/pkg/util/mocks/dynamic"
	"github.com/Azure/ARO-RP/pkg/validate/dynamic"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type fakeTokenCredential struct{}

func (c fakeTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	token, err := jwt.New(jwt.SigningMethodHS256).SignedString([]byte{})
	if err != nil {
		return azcore.AccessToken{}, err
	}

	return azcore.AccessToken{Token: token}, nil
}

func TestCheck(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

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
				validator.EXPECT().ValidateServicePrincipal(ctx, &fakeTokenCredential{})
				return validator
			},
		},
		{
			name:             "fake validation on Service Principal error",
			credentialsExist: true,
			validator: func(controller *gomock.Controller) dynamic.ServicePrincipalValidator {
				validator := mock_dynamic.NewMockDynamic(controller)
				validator.EXPECT().ValidateServicePrincipal(ctx, &fakeTokenCredential{}).
					Return(errors.New("fake validation error"))
				return validator
			},
			wantErr: "fake validation error",
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
				getTokenCredential: func(*azureclient.AROEnvironment) (azcore.TokenCredential, error) {
					return &fakeTokenCredential{}, nil
				},
				newSPValidator: func(azEnv *azureclient.AROEnvironment) dynamic.ServicePrincipalValidator {
					if validatorMock != nil {
						return validatorMock
					}
					return nil
				},
			}

			err := sp.Check(ctx, azure.PublicCloud.Name)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
