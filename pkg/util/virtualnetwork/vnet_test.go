package virtualnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/env"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

func TestCreate(t *testing.T) {
	ctx := context.Background()

	env := &env.Test{
		TestSubscriptionID: "rpSubscriptionId",
		TestResourceGroup:  "rpResourcegroup",
		TestLocation:       "eastus",
	}

	type test struct {
		name  string
		nsgID string
		string
		mocks   func(*test, *mock_network.MockVirtualNetworksClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name:  "valid",
			nsgID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/networkSecurityGroups/rp-pe-nsg",
			mocks: func(tt *test, virtualnetworks *mock_network.MockVirtualNetworksClient) {
				virtualnetworks.EXPECT().
					CreateOrUpdateAndWait(ctx, "rpResourcegroup", "rp-pe-vnet1", mgmtnetwork.VirtualNetwork{
						Name: to.StringPtr("rp-pe-vnet1"),
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							AddressSpace: &mgmtnetwork.AddressSpace{
								AddressPrefixes: &[]string{"10.0.4.0/22"},
							},
							Subnets: &[]mgmtnetwork.Subnet{
								{
									SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
										AddressPrefix: to.StringPtr("10.0.4.0/22"),
										NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
											ID: to.StringPtr("/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/networkSecurityGroups/rp-pe-nsg"),
										},
									},
								},
							},
						},
						Location: to.StringPtr("eastus"),
					}).
					Return(nil)
			},
		},
		{
			name:  "internal error",
			nsgID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/networkSecurityGroups/rp-pe-nsg",
			mocks: func(tt *test, virtualnetworks *mock_network.MockVirtualNetworksClient) {
				virtualnetworks.EXPECT().
					CreateOrUpdateAndWait(ctx, "rpResourcegroup", "rp-pe-vnet1", mgmtnetwork.VirtualNetwork{
						Name: to.StringPtr("rp-pe-vnet1"),
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							AddressSpace: &mgmtnetwork.AddressSpace{
								AddressPrefixes: &[]string{"10.0.4.0/22"},
							},
							Subnets: &[]mgmtnetwork.Subnet{
								{
									SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
										AddressPrefix: to.StringPtr("10.0.4.0/22"),
										NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
											ID: to.StringPtr("/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/networkSecurityGroups/rp-pe-nsg"),
										},
									},
								},
							},
						},
						Location: to.StringPtr("eastus"),
					}).
					Return(fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			virtualnetworks := mock_network.NewMockVirtualNetworksClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, virtualnetworks)
			}

			m := &manager{
				env:             env,
				virtualnetworks: virtualnetworks,
			}

			err := m.Create(ctx, "vnet1", "10.0.4.0/22", tt.nsgID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	env := &env.Test{
		TestResourceGroup: "rpResourcegroup",
	}

	type test struct {
		name    string
		nsgID   string
		mocks   func(*test, *mock_network.MockVirtualNetworksClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name:  "valid",
			nsgID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/networkSecurityGroups/rp-pe-nsg",
			mocks: func(tt *test, virtualnetworks *mock_network.MockVirtualNetworksClient) {
				virtualnetworks.EXPECT().
					DeleteAndWait(ctx, "rpResourcegroup", "rp-pe-vnet1").
					Return(nil)
			},
		},
		{
			name:  "internal error",
			nsgID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/networkSecurityGroups/rp-pe-nsg",
			mocks: func(tt *test, virtualnetworks *mock_network.MockVirtualNetworksClient) {
				virtualnetworks.EXPECT().
					DeleteAndWait(ctx, "rpResourcegroup", "rp-pe-vnet1").
					Return(fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			virtualnetworks := mock_network.NewMockVirtualNetworksClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, virtualnetworks)
			}

			m := &manager{
				env:             env,
				virtualnetworks: virtualnetworks,
			}

			err := m.Delete(ctx, "vnet1")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
