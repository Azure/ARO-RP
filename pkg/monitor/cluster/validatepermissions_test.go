package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_dynamic "github.com/Azure/ARO-RP/pkg/util/mocks/dynamic"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitValidatePermissions(t *testing.T) {
	for _, tt := range []struct {
		name                                string
		mockVnetError                       error // Mock error for ValidateVnet
		mockSubnetError                     error // Mock error for ValidateVnet
		mockValidateDiskEncryptionSetsError error // Mock error for ValidateVnet
		expectedValidateVnet                string
		expectedSubnet                      string
		expectedValidateDiskEncryptionSets  string
	}{
		{
			name:                                "VnetError",
			mockVnetError:                       errors.New("test"), // Mock an error
			mockSubnetError:                     errors.New("test"), // Mock an error
			mockValidateDiskEncryptionSetsError: errors.New("test"), // Mock an error
			expectedValidateVnet:                "test",
			expectedSubnet:                      "test",
			expectedValidateDiskEncryptionSets:  "test",
		},
		{
			name:          "NoError",
			mockVnetError: nil, // No error
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)
			validator := mock_dynamic.NewMockDynamic(controller)
			oc := &api.OpenShiftCluster{}
			mon := &Monitor{
				m:         m,
				oc:        oc,
				validator: validator, // Set the mock validator
			}

			validator.EXPECT().
				ValidateVnet(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tt.mockVnetError)

			validator.EXPECT().
				ValidateSubnets(ctx, gomock.Any(), gomock.Any()).
				Return(tt.mockSubnetError)

			validator.EXPECT().
				ValidateDiskEncryptionSets(ctx, gomock.Any()).
				Return(tt.mockValidateDiskEncryptionSetsError)

			if tt.mockVnetError != nil {
				m.EXPECT().EmitGauge("cluster.validateVnet.permissions", int64(1), map[string]string{
					"vnetError": tt.expectedValidateVnet,
				})
			}

			if tt.mockSubnetError != nil {
				m.EXPECT().EmitGauge("cluster.validateSubnets.permissions", int64(1), map[string]string{
					"subnetError": tt.expectedSubnet,
				})
			}

			if tt.mockValidateDiskEncryptionSetsError != nil {
				m.EXPECT().EmitGauge("cluster.validateDiskEncryptionSets.permissions", int64(1), map[string]string{
					"diskEncryptionSetError": tt.expectedValidateDiskEncryptionSets,
				})
			}
			err := mon.emitValidatePermissions(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
