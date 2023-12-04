package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_dynamic "github.com/Azure/ARO-RP/pkg/util/mocks/dynamic"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

var (
	resourceGroupName = "testGroup"
	subscriptionID    = "0000000-0000-0000-0000-000000000000"
	resourceGroupID   = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroupName
	vnetName          = "testVnet"
	vnetID            = resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/" + vnetName
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
		mocks                               func(*mock_authorization.MockPermissionsClient, context.CancelFunc)
	}{
		{
			name:                 "fail: missing permissions",
			expectedValidateVnet: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have Network Contributor role on vnet '" + vnetID + "'.",
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

			if tt.mockVnetError == nil {
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
