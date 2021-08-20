package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
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
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

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
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestNetworkSecurityGroupID(t *testing.T) {
	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				ResourceGroupID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup",
			},
			MasterProfile: api.MasterProfile{
				SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
				},
			},
		},
	}

	for _, tt := range []struct {
		name        string
		infraID     string
		archVersion api.ArchitectureVersion
		subnetID    string
		wantNSGID   string
		wantErr     string
	}{
		{
			name:      "master arch v1",
			subnetID:  "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			wantNSGID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg",
		},
		{
			name:      "worker arch v1",
			infraID:   "test-1234",
			subnetID:  "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
			wantNSGID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/test-1234-node-nsg",
		},
		{
			name:        "master arch v2",
			archVersion: api.ArchitectureVersionV2,
			subnetID:    "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			wantNSGID:   "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/aro-nsg",
		},
		{
			name:        "worker arch v2",
			infraID:     "test-1234",
			archVersion: api.ArchitectureVersionV2,
			subnetID:    "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
			wantNSGID:   "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/test-1234-nsg",
		},
		{
			name:        "unknown architecture version",
			archVersion: api.ArchitectureVersion(42),
			wantErr:     `unknown architecture version 42`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc.Properties.InfraID = tt.infraID
			oc.Properties.ArchitectureVersion = tt.archVersion

			nsgID, err := NetworkSecurityGroupID(oc, tt.subnetID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if nsgID != tt.wantNSGID {
				t.Error(nsgID)
			}
		})
	}
}
