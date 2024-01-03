package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
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
				Name: to.StringPtr("subnet"),
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

func TestGetGetHighestFreeIP(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name    string
		mocks   func(*test, *mock_network.MockSubnetsClient)
		wantIP  string
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "valid",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					Get(ctx, "vnetResourceGroup", "vnet", "subnet", "ipConfigurations").
					Return(mgmtnetwork.Subnet{
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("10.0.0.0/29"),
						},
					}, nil)
			},
			wantIP: "10.0.0.6",
		},
		{
			name: "valid, use addressPrefixes",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					Get(ctx, "vnetResourceGroup", "vnet", "subnet", "ipConfigurations").
					Return(mgmtnetwork.Subnet{
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefixes: to.StringSlicePtr([]string{"10.0.0.0/29"}),
						},
					}, nil)
			},
			wantIP: "10.0.0.6",
		},
		{
			name: "valid, top address used",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					Get(ctx, "vnetResourceGroup", "vnet", "subnet", "ipConfigurations").
					Return(mgmtnetwork.Subnet{
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("10.0.0.0/29"),
							IPConfigurations: &[]mgmtnetwork.IPConfiguration{
								{
									IPConfigurationPropertiesFormat: &mgmtnetwork.IPConfigurationPropertiesFormat{
										PrivateIPAddress: to.StringPtr("10.0.0.6"),
									},
								},
								{
									IPConfigurationPropertiesFormat: &mgmtnetwork.IPConfigurationPropertiesFormat{},
								},
							},
						},
					}, nil)
			},
			wantIP: "10.0.0.5",
		},
		{
			name: "exhausted",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					Get(ctx, "vnetResourceGroup", "vnet", "subnet", "ipConfigurations").
					Return(mgmtnetwork.Subnet{
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("10.0.0.0/29"),
							IPConfigurations: &[]mgmtnetwork.IPConfiguration{
								{
									IPConfigurationPropertiesFormat: &mgmtnetwork.IPConfigurationPropertiesFormat{
										PrivateIPAddress: to.StringPtr("10.0.0.4"),
									},
								},
								{
									IPConfigurationPropertiesFormat: &mgmtnetwork.IPConfigurationPropertiesFormat{
										PrivateIPAddress: to.StringPtr("10.0.0.5"),
									},
								},
								{
									IPConfigurationPropertiesFormat: &mgmtnetwork.IPConfigurationPropertiesFormat{
										PrivateIPAddress: to.StringPtr("10.0.0.6"),
									},
								},
							},
						},
					}, nil)
			},
		},
		{
			name: "broken",
			mocks: func(tt *test, subnets *mock_network.MockSubnetsClient) {
				subnets.EXPECT().
					Get(ctx, "vnetResourceGroup", "vnet", "subnet", "ipConfigurations").
					Return(mgmtnetwork.Subnet{}, fmt.Errorf("broken"))
			},
			wantErr: "broken",
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

			ip, err := m.GetHighestFreeIP(ctx, "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet")
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if ip != tt.wantIP {
				t.Error(ip)
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
