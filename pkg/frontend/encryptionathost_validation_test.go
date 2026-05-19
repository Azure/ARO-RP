package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"

	mock_frontend "github.com/Azure/ARO-RP/pkg/util/mocks/frontend"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestValidateEncryptionAtHostFeature(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name              string
		fieldPath         string
		mockFeatureState  *string
		mockPropertiesNil bool
		mockStateNil      bool
		mockGetErr        error
		wantErr           string
	}{
		{
			name:             "feature is registered - should return nil",
			fieldPath:        "properties.masterProfile.encryptionAtHost",
			mockFeatureState: pointerutils.ToPtr("Registered"),
			wantErr:          "",
		},
		{
			name:             "feature is not registered - should return 400 BadRequest",
			fieldPath:        "properties.masterProfile.encryptionAtHost",
			mockFeatureState: pointerutils.ToPtr("NotRegistered"),
			wantErr:          "400: InvalidParameter: properties.masterProfile.encryptionAtHost: Microsoft.Compute/EncryptionAtHost feature is not registered on subscription test-subscription. Please register the feature on your subscription before creating the cluster.",
		},
		{
			name:              "resp.Properties is nil - should return internal server error",
			fieldPath:         "properties.workerProfiles[0].encryptionAtHost",
			mockPropertiesNil: true,
			wantErr:           "500: InternalServerError: : Microsoft.Compute/EncryptionAtHost feature has no state for subscription test-subscription.",
		},
		{
			name:         "resp.Properties.State is nil - should return internal server error",
			fieldPath:    "properties.masterProfile.encryptionAtHost",
			mockStateNil: true,
			wantErr:      "500: InternalServerError: : Microsoft.Compute/EncryptionAtHost feature has no state for subscription test-subscription.",
		},
		{
			name:       "Get returns error",
			fieldPath:  "properties.masterProfile.encryptionAtHost",
			mockGetErr: errors.New("random error"),
			wantErr:    "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			featuresClient := mock_frontend.NewMockFeaturesClient(controller)

			var mockResponse armfeatures.ClientGetResponse
			if tt.mockPropertiesNil {
				mockResponse = armfeatures.ClientGetResponse{
					FeatureResult: armfeatures.FeatureResult{
						Properties: nil,
					},
				}
			} else if tt.mockStateNil {
				mockResponse = armfeatures.ClientGetResponse{
					FeatureResult: armfeatures.FeatureResult{
						Properties: &armfeatures.FeatureProperties{
							State: nil,
						},
					},
				}
			} else if tt.mockFeatureState != nil {
				mockResponse = armfeatures.ClientGetResponse{
					FeatureResult: armfeatures.FeatureResult{
						Properties: &armfeatures.FeatureProperties{
							State: tt.mockFeatureState,
						},
					},
				}
			}

			featuresClient.EXPECT().
				Get(ctx, "Microsoft.Compute", "EncryptionAtHost", nil).
				Return(mockResponse, tt.mockGetErr)

			err := validateEncryptionAtHostFeature(ctx, featuresClient, "test-subscription", tt.fieldPath)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("expected error %q, got %q", tt.wantErr, err)
			}
		})
	}
}
