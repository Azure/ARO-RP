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

//
//func TestGetIP(t *testing.T) {
//	ctx := context.Background()
//
//	doc := &api.OpenShiftClusterDocument{
//		ID: "id",
//	}
//
//	env := &env.Test{
//		TestResourceGroup: "rpResourcegroup",
//	}
//
//	type test struct {
//		name     string
//		subnetID string
//		mocks    func(*test, *mock_network.MockPrivateEndpointsClient)
//		wantIP   string
//		wantErr  string
//	}
//
//	for _, tt := range []*test{
//		{
//			name:     "valid",
//			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
//			mocks: func(tt *test, privateendpoints *mock_network.MockPrivateEndpointsClient) {
//				privateendpoints.EXPECT().
//					Get(ctx, "rpResourcegroup", "rp-pe-id", "networkInterfaces").
//					Return(mgmtnetwork.PrivateEndpoint{
//						PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
//							NetworkInterfaces: &[]mgmtnetwork.Interface{
//								{
//									InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
//										IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
//											{
//												InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
//													PrivateIPAddress: &tt.wantIP,
//												},
//											},
//										},
//									},
//								},
//							},
//						},
//					}, nil)
//			},
//			wantIP: "1.2.3.4",
//		},
//		{
//			name:     "internal error",
//			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
//			mocks: func(tt *test, privateendpoints *mock_network.MockPrivateEndpointsClient) {
//				privateendpoints.EXPECT().
//					Get(ctx, "rpResourcegroup", "rp-pe-id", "networkInterfaces").
//					Return(mgmtnetwork.PrivateEndpoint{}, fmt.Errorf("random error"))
//			},
//			wantErr: "random error",
//		},
//	} {
//		t.Run(tt.name, func(t *testing.T) {
//			controller := gomock.NewController(t)
//			defer controller.Finish()
//
//			privateendpoints := mock_network.NewMockPrivateEndpointsClient(controller)
//			if tt.mocks != nil {
//				tt.mocks(tt, privateendpoints)
//			}
//
//			m := &manager{
//				env:              env,
//				privateendpoints: privateendpoints,
//			}
//
//			ip, err := m.GetIP(ctx, doc)
//			if err != nil && err.Error() != tt.wantErr ||
//				err == nil && tt.wantErr != "" {
//				t.Error(err)
//			}
//
//			if ip != tt.wantIP {
//				t.Error(ip)
//			}
//		})
//	}
//}
//
