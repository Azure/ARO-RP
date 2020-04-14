package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mockfeatures "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mocknetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

func TestValidateProviders(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	dv := &openShiftClusterDynamicValidator{}

	for _, tt := range []struct {
		name    string
		mocks   func(*mockfeatures.MockProvidersClient)
		wantErr string
	}{
		{
			name: "all registered",
			mocks: func(providersClient *mockfeatures.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return([]mgmtfeatures.Provider{
						{
							Namespace:         to.StringPtr("Microsoft.Authorization"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Compute"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Network"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Storage"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherRegisteredProvider"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherNotRegisteredProvider"),
							RegistrationState: to.StringPtr("NotRegistered"),
						},
					}, nil)
			},
		},
		{
			name: "compute not registered",
			mocks: func(providersClient *mockfeatures.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return([]mgmtfeatures.Provider{
						{
							Namespace:         to.StringPtr("Microsoft.Authorization"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Compute"),
							RegistrationState: to.StringPtr("NotRegistered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Network"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Storage"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherRegisteredProvider"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherNotRegisteredProvider"),
							RegistrationState: to.StringPtr("NotRegistered"),
						},
					}, nil)
			},
			wantErr: "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Compute' is not registered.",
		},
		{
			name: "storage missing",
			mocks: func(providersClient *mockfeatures.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return([]mgmtfeatures.Provider{
						{
							Namespace:         to.StringPtr("Microsoft.Authorization"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Compute"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Network"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherRegisteredProvider"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherNotRegisteredProvider"),
							RegistrationState: to.StringPtr("NotRegistered"),
						},
					}, nil)
			},
			wantErr: "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Storage' is not registered.",
		},
		{
			name: "error case",
			mocks: func(providersClient *mockfeatures.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return(nil, errors.New("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			providerClient := mockfeatures.NewMockProvidersClient(controller)

			tt.mocks(providerClient)

			err := dv.validateProviders(ctx, providerClient)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateVnet(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	resourceGroup := "testGroup"
	subnetName := "testSubnet"
	vnetName := "testVnet"
	validSubnet := fmt.Sprintf("/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnet/%s", resourceGroup, vnetName, subnetName)

	dv := &openShiftClusterDynamicValidator{}

	for _, tt := range []struct {
		name    string
		mocks   func(*mocknetwork.MockVirtualNetworksClient)
		oc      *api.OpenShiftCluster
		wantErr string
	}{
		{
			name: "fail: custom dns set",
			mocks: func(networksClient *mocknetwork.MockVirtualNetworksClient) {
				networksClient.EXPECT().
					Get(gomock.Any(), resourceGroup, vnetName, "").
					Return(mgmtnetwork.VirtualNetwork{
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							DhcpOptions: &mgmtnetwork.DhcpOptions{
								DNSServers: &[]string{
									"172.16.1.1",
								},
							},
						},
					}, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided vnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet' is invalid: custom DNS servers are not supported.",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: validSubnet,
					},
				},
			},
		},
		{
			name: "pass: default settings",
			mocks: func(networksClient *mocknetwork.MockVirtualNetworksClient) {
				networksClient.EXPECT().
					Get(gomock.Any(), resourceGroup, vnetName, "").
					Return(mgmtnetwork.VirtualNetwork{
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							DhcpOptions: &mgmtnetwork.DhcpOptions{
								DNSServers: &[]string{},
							},
						},
					}, nil)
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: validSubnet,
					},
				},
			},
		},
		{
			name: "fail: vnet not found",
			mocks: func(networksClient *mocknetwork.MockVirtualNetworksClient) {
				networksClient.EXPECT().
					Get(gomock.Any(), resourceGroup, vnetName, "").
					Return(mgmtnetwork.VirtualNetwork{}, fmt.Errorf("The Resource 'Microsoft.Network/virtualNetworks/testVnet' under resource group 'testGroup' was not found."))
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: validSubnet,
					},
				},
			},
			wantErr: "The Resource 'Microsoft.Network/virtualNetworks/testVnet' under resource group 'testGroup' was not found.",
		},
		{
			name: "fail: invalid subnet",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: "/invalid/subnet",
					},
				},
			},
			wantErr: `subnet ID "/invalid/subnet" has incorrect length`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			networkClient := mocknetwork.NewMockVirtualNetworksClient(controller)

			if tt.mocks != nil {
				tt.mocks(networkClient)
			}

			err := dv.validateVnet(ctx, networkClient, tt.oc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
