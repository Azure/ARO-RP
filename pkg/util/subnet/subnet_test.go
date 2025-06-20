package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestGet(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name       string
		subnetID   string
		mocks      func(*test, *mock_network.MockSubnetsClient)
		wantSubnet *mgmtnetwork.Subnet
		wantErr    string
	}

	for _, tt := range []*test{
		{
			name:     "valid",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					Get(ctx, "vnetResourceGroup", "vnet", "subnet", "").
					Return(*tt.wantSubnet, nil)
			},
			wantSubnet: &mgmtnetwork.Subnet{
				Name: pointerutils.ToPtr("subnet"),
			},
		},
		{
			name:     "internal error",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					Get(ctx, "vnetResourceGroup", "vnet", "subnet", "").
					Return(mgmtnetwork.Subnet{}, fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
		{
			name:     "path error",
			subnetID: "/invalid/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			wantErr:  "parsing failed for /invalid/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet. Invalid resource Id format",
		},
		{
			name:     "invalid",
			subnetID: "invalid",
			wantErr:  `subnet ID "invalid" has incorrect length`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnets := mock_network.NewMockSubnetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, subnets)
			}

			m := &manager{
				subnets: subnets,
			}

			subnet, err := m.Get(ctx, tt.subnetID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if !reflect.DeepEqual(subnet, tt.wantSubnet) {
				t.Error(subnet)
			}
		})
	}
}

func TestCreateOrUpdate(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name     string
		subnetID string
		mocks    func(*test, *mock_network.MockSubnetsClient)
		wantErr  string
	}

	for _, tt := range []*test{
		{
			name:     "valid",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					CreateOrUpdateAndWait(ctx, "vnetResourceGroup", "vnet", "subnet", mgmtnetwork.Subnet{}).
					Return(nil)
			},
		},
		{
			name:     "internal error",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					CreateOrUpdateAndWait(ctx, "vnetResourceGroup", "vnet", "subnet", mgmtnetwork.Subnet{}).
					Return(fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
		{
			name:     "path error",
			subnetID: "/invalid/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			wantErr:  "parsing failed for /invalid/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet. Invalid resource Id format",
		},
		{
			name:     "invalid",
			subnetID: "invalid",
			wantErr:  `subnet ID "invalid" has incorrect length`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnets := mock_network.NewMockSubnetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, subnets)
			}

			m := &manager{
				subnets: subnets,
			}

			err := m.CreateOrUpdate(ctx, tt.subnetID, &mgmtnetwork.Subnet{})
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
