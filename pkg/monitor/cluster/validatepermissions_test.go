package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_dynamic "github.com/Azure/ARO-RP/pkg/util/mocks/dynamic"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	"github.com/Azure/ARO-RP/pkg/validate/dynamic"
)

var (
	resourceGroupName = "testGroup"
	subscriptionID    = "0000000-0000-0000-0000-000000000000"
	resourceGroupID   = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroupName
	vnetName          = "testVnet"
	vnetID            = resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/" + vnetName
	masterSubnet      = vnetID + "/subnet/masterSubnet"
	workerSubnet      = vnetID + "/subnet/workerSubnet"
	masterSubnetPath  = "properties.masterProfile.subnetId"
	workerSubnetPath  = "properties.workerProfile.subnetId"
	masterRtID        = resourceGroupID + "/providers/Microsoft.Network/routeTables/masterRt"
	workerRtID        = resourceGroupID + "/providers/Microsoft.Network/routeTables/workerRt"
	masterNgID        = resourceGroupID + "/providers/Microsoft.Network/natGateways/masterNg"
	workerNgID        = resourceGroupID + "/providers/Microsoft.Network/natGateways/workerNg"
	masterNSGv1       = resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg"
	workerNSGv1       = resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg"
)

func TestEmitValidatePermissions(t *testing.T) {
	ctx := context.Background()
	var (
		subscriptionID = "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"
	)
	for _, tt := range []struct {
		name          string
		expectedState error
		PodCIDR       string
		location      string
		ServiceCIDR   string
		subnet        []dynamic.Subnet
		SubnetId      string
		validator     func(controller *gomock.Controller) dynamic.Dynamic
	}{
		{
			name:          "pass",
			location:      "eastus",
			PodCIDR:       "10.128.0.0/14",
			ServiceCIDR:   "0.0.0.0/0",
			subnet:        []dynamic.Subnet{},
			SubnetId:      fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID),
			expectedState: nil,
			validator: func(controller *gomock.Controller) dynamic.Dynamic {
				validator := mock_dynamic.NewMockDynamic(controller)
				validator.EXPECT().ValidateVnet(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				return validator
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			var validatorMock dynamic.Dynamic
			//	fmt.Println(err)
			if tt.validator != nil {
				validatorMock = tt.validator(controller)
			}

			m := mock_metrics.NewMockEmitter(controller)
			/*oc := &api.OpenShiftCluster{
				Location: tt.location,
				Properties: api.OpenShiftClusterProperties{
					NetworkProfile: api.NetworkProfile{
						PodCIDR:     tt.PodCIDR,
						ServiceCIDR: tt.ServiceCIDR,
					},
					MasterProfile: api.MasterProfile{
						SubnetID: tt.SubnetId,
					},
				},
			}*/
			oc := &api.OpenShiftCluster{}

			mon := &Monitor{
				m:         m,
				oc:        oc,
				validator: validatorMock,
			}

			err := validatorMock.ValidateVnet(ctx, tt.location, tt.subnet, tt.ServiceCIDR, tt.PodCIDR)
			fmt.Println(err.Error())
			os.Exit(-1)

			m.EXPECT().EmitGauge("cluster.validateVnet.permissions", int64(1), map[string]string{
				"vnetError": "nil",
			})

			err1 := mon.emitValidatePermissions(ctx)
			if err1 != nil {
				t.Fatal(err1)
			}
		})
	}
}
