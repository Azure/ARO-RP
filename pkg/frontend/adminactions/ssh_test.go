package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestGetNetworkInterfaceSSHInfo(t *testing.T) {
	ctx := context.Background()
	const (
		resourceGroup = "cluster-rg"
		nicName       = "master-nic"
		nicID         = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Network/networkInterfaces/master-nic"
		subnetID      = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master"
		poolID        = "/loadBalancers/lb/backendAddressPools/ssh-1"
	)

	for _, tt := range []struct {
		name    string
		mock    func(*mock_armnetwork.MockInterfacesClient)
		want    *NetworkInterfaceSSHInfo
		wantErr string
	}{
		{
			name: "returns valid pool IDs",
			mock: func(client *mock_armnetwork.MockInterfacesClient) {
				client.EXPECT().Get(gomock.Any(), resourceGroup, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{
					Interface: armnetwork.Interface{ID: pointerutils.ToPtr(nicID), Properties: &armnetwork.InterfacePropertiesFormat{
						IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
							{Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
								Subnet: &armnetwork.Subnet{ID: pointerutils.ToPtr(subnetID)},
								LoadBalancerBackendAddressPools: []*armnetwork.BackendAddressPool{
									{ID: pointerutils.ToPtr(poolID)},
								},
							}},
						},
					}},
				}, nil)
			},
			want: &NetworkInterfaceSSHInfo{ID: nicID, BackendPoolIDs: []string{poolID}, SubnetIDs: []string{subnetID}},
		},
		{
			name: "allows empty pool list",
			mock: func(client *mock_armnetwork.MockInterfacesClient) {
				client.EXPECT().Get(gomock.Any(), resourceGroup, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{
					Interface: armnetwork.Interface{ID: pointerutils.ToPtr(nicID), Properties: &armnetwork.InterfacePropertiesFormat{
						IPConfigurations: []*armnetwork.InterfaceIPConfiguration{{
							Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{Subnet: &armnetwork.Subnet{ID: pointerutils.ToPtr(subnetID)}},
						}},
					}},
				}, nil)
			},
			want: &NetworkInterfaceSSHInfo{ID: nicID, BackendPoolIDs: []string{}, SubnetIDs: []string{subnetID}},
		},
		{
			name: "rejects nil NIC properties",
			mock: func(client *mock_armnetwork.MockInterfacesClient) {
				client.EXPECT().Get(gomock.Any(), resourceGroup, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{Interface: armnetwork.Interface{ID: pointerutils.ToPtr(nicID)}}, nil)
			},
			wantErr: `network interface "master-nic" has nil properties`,
		},
		{
			name: "rejects nil IP configuration",
			mock: func(client *mock_armnetwork.MockInterfacesClient) {
				client.EXPECT().Get(gomock.Any(), resourceGroup, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{
					Interface: armnetwork.Interface{ID: pointerutils.ToPtr(nicID), Properties: &armnetwork.InterfacePropertiesFormat{
						IPConfigurations: []*armnetwork.InterfaceIPConfiguration{nil},
					}},
				}, nil)
			},
			wantErr: `network interface "master-nic" has nil IP configuration at index 0`,
		},
		{
			name: "rejects nil IP configuration properties",
			mock: func(client *mock_armnetwork.MockInterfacesClient) {
				client.EXPECT().Get(gomock.Any(), resourceGroup, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{
					Interface: armnetwork.Interface{ID: pointerutils.ToPtr(nicID), Properties: &armnetwork.InterfacePropertiesFormat{
						IPConfigurations: []*armnetwork.InterfaceIPConfiguration{{}},
					}},
				}, nil)
			},
			wantErr: `network interface "master-nic" IP configuration at index 0 has nil properties`,
		},
		{
			name: "rejects nil backend pool",
			mock: func(client *mock_armnetwork.MockInterfacesClient) {
				client.EXPECT().Get(gomock.Any(), resourceGroup, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{
					Interface: armnetwork.Interface{ID: pointerutils.ToPtr(nicID), Properties: &armnetwork.InterfacePropertiesFormat{
						IPConfigurations: []*armnetwork.InterfaceIPConfiguration{{
							Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
								Subnet:                          &armnetwork.Subnet{ID: pointerutils.ToPtr(subnetID)},
								LoadBalancerBackendAddressPools: []*armnetwork.BackendAddressPool{nil},
							},
						}},
					}},
				}, nil)
			},
			wantErr: `network interface "master-nic" IP configuration at index 0 has nil backend pool at index 0`,
		},
		{
			name: "rejects nil backend pool ID",
			mock: func(client *mock_armnetwork.MockInterfacesClient) {
				client.EXPECT().Get(gomock.Any(), resourceGroup, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{
					Interface: armnetwork.Interface{ID: pointerutils.ToPtr(nicID), Properties: &armnetwork.InterfacePropertiesFormat{
						IPConfigurations: []*armnetwork.InterfaceIPConfiguration{{
							Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
								Subnet:                          &armnetwork.Subnet{ID: pointerutils.ToPtr(subnetID)},
								LoadBalancerBackendAddressPools: []*armnetwork.BackendAddressPool{{}},
							},
						}},
					}},
				}, nil)
			},
			wantErr: `network interface "master-nic" IP configuration at index 0 backend pool at index 0 has nil ID`,
		},
		{
			name: "propagates Azure error",
			mock: func(client *mock_armnetwork.MockInterfacesClient) {
				client.EXPECT().Get(gomock.Any(), resourceGroup, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{}, errors.New("azure boom"))
			},
			wantErr: "azure boom",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			client := mock_armnetwork.NewMockInterfacesClient(controller)
			tt.mock(client)

			a := &azureActions{networkInterfaces: client}
			got, err := a.GetNetworkInterfaceSSHInfo(ctx, resourceGroup, nicName)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("got error %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSSHRouteStatus(t *testing.T) {
	const (
		resourceGroup          = "ipConfigurations"
		clusterResourceGroupID = "/subscriptions/sub/resourceGroups/ipConfigurations"
		loadBalancer           = "infra-internal"
		loadBalancerID         = clusterResourceGroupID + "/providers/Microsoft.Network/loadBalancers/infra-internal"
		backendPoolID          = loadBalancerID + "/backendAddressPools/ssh-1"
		frontendID             = loadBalancerID + "/frontendIPConfigurations/private"
		otherFrontendID        = loadBalancerID + "/frontendIPConfigurations/other"
		frontendIP             = "10.0.0.4"
		frontendPort           = int32(2201)
		requestedNICID         = clusterResourceGroupID + "/providers/Microsoft.Network/networkInterfaces/master-1-nic"
		otherNICID             = clusterResourceGroupID + "/providers/Microsoft.Network/networkInterfaces/master-2-nic"
	)

	makeLB := func(backendNICIDs []string) armnetwork.LoadBalancer {
		backendIPConfigurations := make([]*armnetwork.InterfaceIPConfiguration, 0, len(backendNICIDs))
		for _, nicID := range backendNICIDs {
			backendIPConfigurations = append(backendIPConfigurations, &armnetwork.InterfaceIPConfiguration{ID: pointerutils.ToPtr(nicID + "/ipConfigurations/ipconfig1")})
		}
		return armnetwork.LoadBalancer{Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{ID: pointerutils.ToPtr(frontendID), Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{PrivateIPAddress: pointerutils.ToPtr(frontendIP)}},
				{ID: pointerutils.ToPtr(otherFrontendID), Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{PrivateIPAddress: pointerutils.ToPtr("10.0.0.5")}},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{{
				ID:         pointerutils.ToPtr(backendPoolID),
				Properties: &armnetwork.BackendAddressPoolPropertiesFormat{BackendIPConfigurations: backendIPConfigurations},
			}},
			LoadBalancingRules: []*armnetwork.LoadBalancingRule{
				{Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
					FrontendIPConfiguration: &armnetwork.SubResource{ID: pointerutils.ToPtr(frontendID)},
					BackendAddressPool:      &armnetwork.SubResource{ID: pointerutils.ToPtr(backendPoolID)},
					FrontendPort:            pointerutils.ToPtr(frontendPort),
					BackendPort:             pointerutils.ToPtr[int32](22),
					Protocol:                pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
				}},
				{Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
					FrontendIPConfiguration: &armnetwork.SubResource{ID: pointerutils.ToPtr(otherFrontendID)},
					BackendAddressPool:      &armnetwork.SubResource{ID: pointerutils.ToPtr(backendPoolID)},
					FrontendPort:            pointerutils.ToPtr(frontendPort),
					BackendPort:             pointerutils.ToPtr[int32](22),
					Protocol:                pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
				}},
			},
		}}
	}

	t.Run("uses authoritative pool and final IP configuration delimiter", func(t *testing.T) {
		controller := gomock.NewController(t)
		loadBalancers := mock_armnetwork.NewMockLoadBalancersClient(controller)
		loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, loadBalancer, nil).Return(armnetwork.LoadBalancersClientGetResponse{
			LoadBalancer: makeLB([]string{requestedNICID, otherNICID}),
		}, nil)

		a := &azureActions{loadBalancers: loadBalancers}
		got, err := a.GetSSHRouteStatus(context.Background(), resourceGroup, clusterResourceGroupID, loadBalancer, backendPoolID, requestedNICID, frontendIP, frontendPort)
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(got.BackendPoolNetworkInterfaceIDs)
		wantIDs := []string{otherNICID, requestedNICID}
		sort.Strings(wantIDs)
		if !reflect.DeepEqual(got.BackendPoolNetworkInterfaceIDs, wantIDs) {
			t.Fatalf("got pool NICs %v, want %v", got.BackendPoolNetworkInterfaceIDs, wantIDs)
		}
		if got.LoadBalancingRuleCount != 1 {
			t.Fatalf("got %d matching rules, want 1", got.LoadBalancingRuleCount)
		}
	})

	t.Run("rejects IP-based backend member", func(t *testing.T) {
		controller := gomock.NewController(t)
		loadBalancers := mock_armnetwork.NewMockLoadBalancersClient(controller)
		loadBalancerResource := makeLB([]string{requestedNICID})
		loadBalancerResource.Properties.BackendAddressPools[0].Properties.LoadBalancerBackendAddresses = []*armnetwork.LoadBalancerBackendAddress{{}}
		loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, loadBalancer, nil).Return(armnetwork.LoadBalancersClientGetResponse{
			LoadBalancer: loadBalancerResource,
		}, nil)

		a := &azureActions{loadBalancers: loadBalancers}
		_, err := a.GetSSHRouteStatus(context.Background(), resourceGroup, clusterResourceGroupID, loadBalancer, backendPoolID, requestedNICID, frontendIP, frontendPort)
		wantErr := fmt.Sprintf("load balancer %q backend pool %q contains IP-based backend members", loadBalancer, backendPoolID)
		if err == nil || err.Error() != wantErr {
			t.Fatalf("got error %v, want %q", err, wantErr)
		}
	})

	t.Run("rejects foreign resource group member", func(t *testing.T) {
		controller := gomock.NewController(t)
		loadBalancers := mock_armnetwork.NewMockLoadBalancersClient(controller)
		foreignNICID := "/subscriptions/sub/resourceGroups/other/providers/Microsoft.Network/networkInterfaces/foreign-nic"
		loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, loadBalancer, nil).Return(armnetwork.LoadBalancersClientGetResponse{
			LoadBalancer: makeLB([]string{requestedNICID, foreignNICID}),
		}, nil)

		a := &azureActions{loadBalancers: loadBalancers}
		_, err := a.GetSSHRouteStatus(context.Background(), resourceGroup, clusterResourceGroupID, loadBalancer, backendPoolID, requestedNICID, frontendIP, frontendPort)
		wantErr := fmt.Sprintf("load balancer %q backend pool %q references NIC %q outside the cluster resource group", loadBalancer, backendPoolID, foreignNICID)
		if err == nil || err.Error() != wantErr {
			t.Fatalf("got error %v, want %q", err, wantErr)
		}
	})
}
