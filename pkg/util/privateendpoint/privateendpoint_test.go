package privateendpoint

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestCreate(t *testing.T) {
	ctx := context.Background()

	doc := &api.OpenShiftClusterDocument{
		ID: "id",
		OpenShiftCluster: &api.OpenShiftCluster{
			Location: "eastus",
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup",
				},
			},
		},
	}

	type test struct {
		name     string
		infraID  string
		subnetID string
		mocks    func(*test, *mock_network.MockPrivateEndpointsClient)
		wantErr  string
	}

	for _, tt := range []*test{
		{
			name:     "valid",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, privateendpoints *mock_network.MockPrivateEndpointsClient) {
				privateendpoints.EXPECT().
					CreateOrUpdateAndWait(ctx, "rpResourcegroup", "rp-pe-id", mgmtnetwork.PrivateEndpoint{
						PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr("/subscriptions/rpSubscriptionId/resourceGroups/rpResourcegroup/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet"),
							},
							ManualPrivateLinkServiceConnections: &[]mgmtnetwork.PrivateLinkServiceConnection{
								{
									Name: to.StringPtr("rp-plsconnection"),
									PrivateLinkServiceConnectionProperties: &mgmtnetwork.PrivateLinkServiceConnectionProperties{
										PrivateLinkServiceID: to.StringPtr("/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/privateLinkServices/aro-pls"),
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
			name:     "internal error",
			infraID:  "test-1234",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, privateendpoints *mock_network.MockPrivateEndpointsClient) {
				privateendpoints.EXPECT().
					CreateOrUpdateAndWait(ctx, "rpResourcegroup", "rp-pe-id", mgmtnetwork.PrivateEndpoint{
						PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr("/subscriptions/rpSubscriptionId/resourceGroups/rpResourcegroup/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet"),
							},
							ManualPrivateLinkServiceConnections: &[]mgmtnetwork.PrivateLinkServiceConnection{
								{
									Name: to.StringPtr("rp-plsconnection"),
									PrivateLinkServiceConnectionProperties: &mgmtnetwork.PrivateLinkServiceConnectionProperties{
										PrivateLinkServiceID: to.StringPtr("/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/privateLinkServices/test-1234-pls"),
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

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().SubscriptionID().AnyTimes().Return("rpSubscriptionId")
			env.EXPECT().ResourceGroup().AnyTimes().Return("rpResourcegroup")

			privateendpoints := mock_network.NewMockPrivateEndpointsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, privateendpoints)
			}

			m := &manager{
				env:              env,
				privateendpoints: privateendpoints,
			}

			doc.OpenShiftCluster.Properties.InfraID = tt.infraID

			err := m.Create(ctx, doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	doc := &api.OpenShiftClusterDocument{
		ID: "id",
	}

	type test struct {
		name     string
		subnetID string
		mocks    func(*test, *mock_network.MockPrivateEndpointsClient)
		wantErr  string
	}

	for _, tt := range []*test{
		{
			name:     "valid",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, privateendpoints *mock_network.MockPrivateEndpointsClient) {
				privateendpoints.EXPECT().
					DeleteAndWait(ctx, "rpResourcegroup", "rp-pe-id").
					Return(nil)
			},
		},
		{
			name:     "internal error",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, privateendpoints *mock_network.MockPrivateEndpointsClient) {
				privateendpoints.EXPECT().
					DeleteAndWait(ctx, "rpResourcegroup", "rp-pe-id").
					Return(fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ResourceGroup().AnyTimes().Return("rpResourcegroup")

			privateendpoints := mock_network.NewMockPrivateEndpointsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, privateendpoints)
			}

			m := &manager{
				env:              env,
				privateendpoints: privateendpoints,
			}

			err := m.Delete(ctx, doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestGetIP(t *testing.T) {
	ctx := context.Background()

	doc := &api.OpenShiftClusterDocument{
		ID: "id",
	}

	type test struct {
		name     string
		subnetID string
		mocks    func(*test, *mock_network.MockPrivateEndpointsClient)
		wantIP   string
		wantErr  string
	}

	for _, tt := range []*test{
		{
			name:     "valid",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, privateendpoints *mock_network.MockPrivateEndpointsClient) {
				privateendpoints.EXPECT().
					Get(ctx, "rpResourcegroup", "rp-pe-id", "networkInterfaces").
					Return(mgmtnetwork.PrivateEndpoint{
						PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
							NetworkInterfaces: &[]mgmtnetwork.Interface{
								{
									InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
										IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
											{
												InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
													PrivateIPAddress: &tt.wantIP,
												},
											},
										},
									},
								},
							},
						},
					}, nil)
			},
			wantIP: "1.2.3.4",
		},
		{
			name:     "internal error",
			subnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet",
			mocks: func(tt *test, privateendpoints *mock_network.MockPrivateEndpointsClient) {
				privateendpoints.EXPECT().
					Get(ctx, "rpResourcegroup", "rp-pe-id", "networkInterfaces").
					Return(mgmtnetwork.PrivateEndpoint{}, fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ResourceGroup().AnyTimes().Return("rpResourcegroup")

			privateendpoints := mock_network.NewMockPrivateEndpointsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, privateendpoints)
			}

			m := &manager{
				env:              env,
				privateendpoints: privateendpoints,
			}

			ip, err := m.GetIP(ctx, doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if ip != tt.wantIP {
				t.Error(ip)
			}
		})
	}
}
