package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_aad "github.com/Azure/ARO-RP/pkg/util/mocks/aad"
	mock_dynamic "github.com/Azure/ARO-RP/pkg/util/mocks/dynamic"
)

func TestOpenshiftDynamicValidate(t *testing.T) {
	errorMessage := "some error"
	tests := []struct {
		name string

		diskValidatorErr error
		encrytionErr     error
		providerError    error
		vmSKUError       error
		vnetError        error
		subnetError      error
		spError          error
		tokenError       error

		oc  *api.OpenShiftCluster
		sub *api.Subscription

		wantError string
	}{
		{name: "all good"},
		{name: "disk err", diskValidatorErr: errors.New(errorMessage), wantError: errorMessage},
		{name: "encryption err", encrytionErr: errors.New(errorMessage), wantError: errorMessage},
		{name: "subnet err", subnetError: errors.New(errorMessage), wantError: errorMessage},
		{name: "vnet err", vnetError: errors.New(errorMessage), wantError: errorMessage},
		{name: "sp err", spError: errors.New(errorMessage), wantError: errorMessage},
		{name: "vmsku err", vmSKUError: errors.New(errorMessage), wantError: errorMessage},
		{name: "provider err", providerError: errors.New(errorMessage), wantError: errorMessage},
		{name: "token err", tokenError: errors.New(errorMessage), wantError: errorMessage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDisk := mock_dynamic.NewMockDiskValidator(controller)
			mockDisk.EXPECT().Validate(gomock.Any(), gomock.Any()).AnyTimes().Return(tt.diskValidatorErr)

			mockEnc := mock_dynamic.NewMockEncryptionAtHostValidator(controller)
			mockEnc.EXPECT().Validate(gomock.Any(), gomock.Any()).AnyTimes().Return(tt.encrytionErr)

			mockProvider := mock_dynamic.NewMockProvidersValidator(controller)
			mockProvider.EXPECT().Validate(gomock.Any()).AnyTimes().Return(tt.providerError)

			mockVMSKU := mock_dynamic.NewMockVMSKUValidator(controller)
			mockVMSKU.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(tt.vmSKUError)

			mockVnet := mock_dynamic.NewMockVnetValidator(controller)
			mockVnet.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(tt.vnetError)

			mockSubnet := mock_dynamic.NewMockSubnetValidator(controller)
			mockSubnet.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(tt.subnetError)

			mockServiceSprincipal := mock_dynamic.NewMockServicePrincipalValidator(controller)
			mockServiceSprincipal.EXPECT().Validate(gomock.Any()).AnyTimes().Return(tt.spError)

			mockToken := mock_aad.NewMockTokenClient(controller)
			mockToken.EXPECT().GetToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, tt.tokenError)

			validator := openShiftClusterDynamicValidator{
				diskValidator:       mockDisk,
				encryptionValidator: mockEnc,
				providersValidator:  mockProvider,
				vmSKUValidator:      mockVMSKU,
				vnetValidator:       mockVnet,
				subnetValidator:     mockSubnet,
				spValidator:         mockServiceSprincipal,

				tokenClient: mockToken,

				oc:              tt.oc,
				subscriptionDoc: &api.SubscriptionDocument{Subscription: tt.sub},
			}

			if tt.sub == nil {
				validator.subscriptionDoc = &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							TenantID: "someID",
						},
					},
				}
			}
			if tt.oc == nil {
				validator.oc = &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						MasterProfile: api.MasterProfile{SubnetID: "someID"},
						WorkerProfiles: []api.WorkerProfile{
							{SubnetID: "someID"},
						},
					},
				}
			}

			err := validator.dynamic(context.Background(), "", "")
			if (err == nil && tt.wantError != "") || (err != nil && err.Error() != tt.wantError) {
				t.Errorf("wanted error to be %s but got %q", tt.wantError, err)
			}
		})
	}
}
