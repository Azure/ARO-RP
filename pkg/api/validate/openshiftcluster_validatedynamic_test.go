package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mockauthorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mockfeatures "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

func TestValidateProviders(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

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

			dv := &openShiftClusterDynamicValidator{
				log:         logrus.NewEntry(logrus.StandardLogger()),
				spProviders: providerClient,
			}

			err := dv.validateProviders(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateVnet(t *testing.T) {
	ctx := context.Background()

	resourceGroupID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup"
	vnetID := resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/testVnet"
	masterSubnet := vnetID + "/subnet/masterSubnet"
	workerSubnet := vnetID + "/subnet/workerSubnet"
	masterNSG := resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg"
	workerNSG := resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg"

	for _, tt := range []struct {
		name       string
		modifyOC   func(*api.OpenShiftCluster)
		modifyVnet func(*mgmtnetwork.VirtualNetwork)
		wantErr    string
	}{
		{
			name: "pass",
		},
		{
			name: "pass (creating)",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup = nil
				(*vnet.Subnets)[1].NetworkSecurityGroup = nil
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
		},
		{
			name: "missing subnet (master)",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				*vnet.Subnets = (*vnet.Subnets)[1:]
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/masterSubnet' could not be found.",
		},
		{
			name: "missing subnet (worker)",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				*vnet.Subnets = (*vnet.Subnets)[:1]
			},
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles["worker"].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' could not be found.`,
		},
		{
			name: "invalid PLS network policy (master)",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].PrivateLinkServiceNetworkPolicies = nil
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/masterSubnet' is invalid: must have privateLinkServiceNetworkPolicies disabled.",
		},
		{
			name: "invalid service endpoint (master)",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].ServiceEndpoints = nil
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/masterSubnet' is invalid: must have Microsoft.ContainerRegistry serviceEndpoint.",
		},
		{
			name: "invalid service endpoint (worker)",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].ServiceEndpoints = nil
			},
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles["worker"].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must have Microsoft.ContainerRegistry serviceEndpoint.`,
		},
		{
			name: "invalid master nsg",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup = nil
			},
			wantErr: `400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/masterSubnet' is invalid: must have network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg' attached.`,
		},
		{
			name: "invalid worker nsg",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].NetworkSecurityGroup = nil
			},
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles["worker"].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must have network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg' attached.`,
		},
		{
			name: "invalid master nsg (creating)",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].NetworkSecurityGroup = nil
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			wantErr: `400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/masterSubnet' is invalid: must not have a network security group attached.`,
		},
		{
			name: "invalid worker nsg (creating)",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup = nil
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles["worker"].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must not have a network security group attached.`,
		},
		{
			name: "invalid master subnet size",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].AddressPrefix = to.StringPtr("10.0.0.0/28")
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/masterSubnet' is invalid: must be /27 or larger.",
		},
		{
			name: "invalid worker subnet size",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].AddressPrefix = to.StringPtr("10.0.0.0/28")
			},
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles["worker"].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must be /27 or larger.`,
		},
		{
			name: "master and worker subnets overlap",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].AddressPrefix = (*vnet.Subnets)[1].AddressPrefix
			},
			wantErr: "400: InvalidLinkedVNet: : The provided CIDRs must not overlap: '10.0.1.0/24 overlaps with 10.0.1.0/24'.",
		},
		{
			name: "master and pod subnets overlap",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].AddressPrefix = to.StringPtr("10.0.2.0/24")
			},
			wantErr: "400: InvalidLinkedVNet: : The provided CIDRs must not overlap: '10.0.2.0/24 overlaps with 10.0.2.0/24'.",
		},
		{
			name: "worker and pod subnets overlap",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].AddressPrefix = to.StringPtr("10.0.2.0/24")
			},
			wantErr: "400: InvalidLinkedVNet: : The provided CIDRs must not overlap: '10.0.2.0/24 overlaps with 10.0.2.0/24'.",
		},
		{
			name: "master and service subnets overlap",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].AddressPrefix = to.StringPtr("10.0.3.0/24")
			},
			wantErr: "400: InvalidLinkedVNet: : The provided CIDRs must not overlap: '10.0.3.0/24 overlaps with 10.0.3.0/24'.",
		},
		{
			name: "worker and service subnets overlap",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].AddressPrefix = to.StringPtr("10.0.3.0/24")
			},
			wantErr: "400: InvalidLinkedVNet: : The provided CIDRs must not overlap: '10.0.3.0/24 overlaps with 10.0.3.0/24'.",
		},
		{
			name: "custom dns set",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				vnet.DhcpOptions.DNSServers = &[]string{
					"172.16.1.1",
				}
			},
			wantErr: "400: InvalidLinkedVNet: : The provided vnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet' is invalid: custom DNS servers are not supported.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
					NetworkProfile: api.NetworkProfile{
						PodCIDR:     "10.0.2.0/24",
						ServiceCIDR: "10.0.3.0/24",
					},
					MasterProfile: api.MasterProfile{
						SubnetID: masterSubnet,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							SubnetID: workerSubnet,
						},
					},
				},
			}

			vnet := &mgmtnetwork.VirtualNetwork{
				ID: &vnetID,
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					DhcpOptions: &mgmtnetwork.DhcpOptions{
						DNSServers: &[]string{},
					},
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								AddressPrefix: to.StringPtr("10.0.0.0/24"),
								NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
									ID: &masterNSG,
								},
								ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
									{
										Service:           to.StringPtr("Microsoft.ContainerRegistry"),
										ProvisioningState: mgmtnetwork.Succeeded,
									},
								},
								PrivateLinkServiceNetworkPolicies: to.StringPtr("Disabled"),
							},
						},
						{
							ID: &workerSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								AddressPrefix: to.StringPtr("10.0.1.0/24"),
								NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
									ID: &workerNSG,
								},
								ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
									{
										Service:           to.StringPtr("Microsoft.ContainerRegistry"),
										ProvisioningState: mgmtnetwork.Succeeded,
									},
								},
							},
						},
					},
				},
			}

			if tt.modifyOC != nil {
				tt.modifyOC(oc)
			}
			if tt.modifyVnet != nil {
				tt.modifyVnet(vnet)
			}

			dv := &openShiftClusterDynamicValidator{
				log: logrus.NewEntry(logrus.StandardLogger()),
				oc:  oc,
			}

			err := dv.validateVnet(ctx, vnet)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateVnetPermissions(t *testing.T) {
	ctx := context.Background()

	resourceGroupID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup"
	vnetID := resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/testVnet"

	controller := gomock.NewController(t)
	defer controller.Finish()

	dv := &openShiftClusterDynamicValidator{
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	for _, tt := range []struct {
		name    string
		mocks   func(*mockauthorization.MockPermissionsClient, func())
		wantErr string
	}{
		{
			name: "pass",
			mocks: func(permissionsClient *mockauthorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), "", "", "", "", "").
					Return([]mgmtauthorization.Permission{
						{
							Actions: &[]string{
								"Microsoft.Network/virtualNetworks/subnets/join/action",
								"Microsoft.Network/virtualNetworks/subnets/read",
								"Microsoft.Network/virtualNetworks/subnets/write",
							},
							NotActions: &[]string{},
						},
					}, nil)
			},
		},
		{
			name: "fail: missing permissions",
			mocks: func(permissionsClient *mockauthorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), "", "", "", "", "").
					Do(func(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) {
						cancel()
					}).
					Return(
						[]mgmtauthorization.Permission{
							{
								Actions:    &[]string{},
								NotActions: &[]string{},
							},
						},
						nil,
					)
			},
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider does not have Contributor permission on vnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet'.",
		},
		{
			name: "fail: not found",
			mocks: func(permissionsClient *mockauthorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), "", "", "", "", "").
					Do(func(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) {
						cancel()
					}).
					Return(
						nil,
						autorest.DetailedError{
							StatusCode: http.StatusNotFound,
						},
					)
			},
			wantErr: "400: InvalidLinkedVNet: : The vnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet' could not be found.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			permissionsClient := mockauthorization.NewMockPermissionsClient(controller)

			tt.mocks(permissionsClient, cancel)

			err := dv.validateVnetPermissions(ctx, &refreshable.TestAuthorizer{}, permissionsClient, vnetID, &azure.Resource{}, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateRouteTablePermissionsSubnet(t *testing.T) {
	ctx := context.Background()

	resourceGroupID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup"
	vnetID := resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/testVnet"
	masterSubnet := vnetID + "/subnet/masterSubnet"
	rtID := resourceGroupID + "/providers/Microsoft.Network/routeTables/testRT"

	controller := gomock.NewController(t)
	defer controller.Finish()

	dv := &openShiftClusterDynamicValidator{
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	for _, tt := range []struct {
		name    string
		mocks   func(*mockauthorization.MockPermissionsClient, func())
		vnet    func(*mgmtnetwork.VirtualNetwork)
		subnet  string
		wantErr string
	}{
		{
			name: "pass",
			mocks: func(permissionsClient *mockauthorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), "testGroup", "Microsoft.Network", "", "routeTables", "testRT").
					Return([]mgmtauthorization.Permission{
						{
							Actions: &[]string{
								"Microsoft.Network/routeTables/join/action",
								"Microsoft.Network/routeTables/read",
								"Microsoft.Network/routeTables/write",
							},
							NotActions: &[]string{},
						},
					}, nil)
			},
			subnet: masterSubnet,
		},
		{
			name: "pass (no route table)",
			vnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].RouteTable = nil
			},
			subnet: masterSubnet,
		},
		{
			name: "fail: missing permissions",
			mocks: func(permissionsClient *mockauthorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), "testGroup", "Microsoft.Network", "", "routeTables", "testRT").
					Do(func(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) {
						cancel()
					}).
					Return(
						[]mgmtauthorization.Permission{
							{
								Actions:    &[]string{},
								NotActions: &[]string{},
							},
						},
						nil,
					)
			},
			subnet:  masterSubnet,
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider does not have Contributor permission on route table '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/routeTables/testRT'.",
		},
		{
			name: "fail: not found",
			mocks: func(permissionsClient *mockauthorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), "testGroup", "Microsoft.Network", "", "routeTables", "testRT").
					Do(func(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) {
						cancel()
					}).
					Return(
						nil,
						autorest.DetailedError{
							StatusCode: http.StatusNotFound,
						},
					)
			},
			subnet:  masterSubnet,
			wantErr: "400: InvalidLinkedRouteTable: : The route table '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/routeTables/testRT' could not be found.",
		},
		{
			name:    "fail: subnet not found",
			subnet:  "doesnotexist",
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The subnet 'doesnotexist' could not be found.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			vnet := &mgmtnetwork.VirtualNetwork{
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								RouteTable: &mgmtnetwork.RouteTable{
									ID: &rtID,
								},
							},
						},
					},
				},
			}

			permissionsClient := mockauthorization.NewMockPermissionsClient(controller)

			if tt.mocks != nil {
				tt.mocks(permissionsClient, cancel)
			}

			if tt.vnet != nil {
				tt.vnet(vnet)
			}

			err := dv.validateRouteTablePermissionsSubnet(ctx, &refreshable.TestAuthorizer{}, permissionsClient, vnet, tt.subnet, "properties.masterProfile.subnetId", api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
