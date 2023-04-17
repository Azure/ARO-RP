package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	mock_remotepdp "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/authz/remotepdp"
	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
	masterRtID        = resourceGroupID + "/providers/Microsoft.Network/routeTables/masterRt"
	workerRtID        = resourceGroupID + "/providers/Microsoft.Network/routeTables/workerRt"
	masterNgID        = resourceGroupID + "/providers/Microsoft.Network/natGateways/masterNg"
	workerNgID        = resourceGroupID + "/providers/Microsoft.Network/natGateways/workerNg"
	masterNSGv1       = resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg"
	workerNSGv1       = resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg"
)

func TestValidateVnetPermissions(t *testing.T) {
	ctx := context.Background()

	resourceType := "virtualNetworks"
	resourceProvider := "Microsoft.Network"

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_authorization.MockPermissionsClient, context.CancelFunc)
		wantErr string
	}{
		{
			name: "pass",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel context.CancelFunc) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, resourceProvider, "", resourceType, vnetName).
					Return([]mgmtauthorization.Permission{
						{
							Actions: &[]string{
								"Microsoft.Network/virtualNetworks/join/action",
								"Microsoft.Network/virtualNetworks/read",
								"Microsoft.Network/virtualNetworks/write",
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
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel context.CancelFunc) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, resourceProvider, "", resourceType, vnetName).
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
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have Network Contributor role on vnet '" + vnetID + "'.",
		},
		{
			name: "fail: not found",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel context.CancelFunc) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, resourceProvider, "", resourceType, vnetName).
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
			wantErr: "400: InvalidLinkedVNet: : The vnet '" + vnetID + "' could not be found.",
		},
		{
			name: "fail: fp/sp has no permission on the target vnet (forbidden error)",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel context.CancelFunc) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, resourceProvider, "", resourceType, vnetName).
					Do(func(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) {
						cancel()
					}).
					Return(
						nil,
						autorest.DetailedError{
							StatusCode: http.StatusForbidden,
							Message:    "some forbidden error on the resource.",
						},
					)
			},

			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have Network Contributor role on vnet '" + vnetID + "'.\nOriginal error message: some forbidden error on the resource.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			permissionsClient := mock_authorization.NewMockPermissionsClient(controller)
			tt.mocks(permissionsClient, cancel)

			dv := &dynamic{
				appID:          "fff51942-b1f9-4119-9453-aaa922259eb7",
				authorizerType: AuthorizerClusterServicePrincipal,
				log:            logrus.NewEntry(logrus.StandardLogger()),
				permissions:    permissionsClient,
			}

			vnetr, err := azure.ParseResourceID(vnetID)
			if err != nil {
				t.Fatal(err)
			}

			err = dv.validateVnetPermissions(ctx, vnetr)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestGetRouteTableID(t *testing.T) {
	for _, tt := range []struct {
		name       string
		modifyVnet func(*mgmtnetwork.VirtualNetwork)
		wantErr    string
	}{
		{
			name: "pass",
		},
		{
			name: "pass: no route table",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].RouteTable = nil
			},
		},
		{
			name: "fail: can't find subnet",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vnet := &mgmtnetwork.VirtualNetwork{
				ID: &vnetID,
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								RouteTable: &mgmtnetwork.RouteTable{
									ID: &masterRtID,
								},
							},
						},
					},
				},
			}

			if tt.modifyVnet != nil {
				tt.modifyVnet(vnet)
			}

			// purposefully hardcoding path to "" so it is not needed in the wantErr message
			_, err := getRouteTableID(vnet, masterSubnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestGetNatGatewayID(t *testing.T) {
	for _, tt := range []struct {
		name       string
		modifyVnet func(*mgmtnetwork.VirtualNetwork)
		wantErr    string
	}{
		{
			name: "pass",
		},
		{
			name: "pass: no nat gateway",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NatGateway = nil
			},
		},
		{
			name: "fail: can't find subnet",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vnet := &mgmtnetwork.VirtualNetwork{
				ID: &vnetID,
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								NatGateway: &mgmtnetwork.SubResource{
									ID: &masterNgID,
								},
							},
						},
					},
				},
			}

			if tt.modifyVnet != nil {
				tt.modifyVnet(vnet)
			}

			// purposefully hardcoding path to "" so it is not needed in the wantErr message
			_, err := getNatGatewayID(vnet, masterSubnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateRouteTablesPermissions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name            string
		subnet          Subnet
		permissionMocks func(*mock_authorization.MockPermissionsClient, context.CancelFunc)
		vnetMocks       func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr         string
	}{
		{
			name:   "fail: failed to get vnet",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, errors.New("failed to get vnet"))
			},
			wantErr: "failed to get vnet",
		},
		{
			name:   "fail: master subnet doesn't exist",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name:   "fail: worker subnet ID doesn't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].ID = to.StringPtr("not valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + workerSubnet + "' could not be found.",
		},
		{
			name:   "fail: permissions don't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			permissionMocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel context.CancelFunc) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, "Microsoft.Network", "", "routeTables", gomock.Any()).
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
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal does not have Network Contributor role on route table '" + workerRtID + "'.",
		},
		{
			name:   "pass",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			permissionMocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel context.CancelFunc) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, "Microsoft.Network", "", "routeTables", gomock.Any()).
					AnyTimes().
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
		},
		{
			name:   "pass: no route table to check",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].RouteTable = nil
				(*vnet.Subnets)[1].RouteTable = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			permissionsClient := mock_authorization.NewMockPermissionsClient(controller)
			vnetClient := mock_network.NewMockVirtualNetworksClient(controller)

			vnet := &mgmtnetwork.VirtualNetwork{
				ID: &vnetID,
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								RouteTable: &mgmtnetwork.RouteTable{
									ID: &masterRtID,
								},
							},
						},
						{
							ID: &workerSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								RouteTable: &mgmtnetwork.RouteTable{
									ID: &workerRtID,
								},
							},
						},
					},
				},
			}

			dv := &dynamic{
				authorizerType:  AuthorizerClusterServicePrincipal,
				log:             logrus.NewEntry(logrus.StandardLogger()),
				permissions:     permissionsClient,
				virtualNetworks: vnetClient,
			}

			if tt.permissionMocks != nil {
				tt.permissionMocks(permissionsClient, cancel)
			}

			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, *vnet)
			}

			err := dv.validateRouteTablePermissions(ctx, tt.subnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateNatGatewaysPermissions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name            string
		subnet          Subnet
		permissionMocks func(*mock_authorization.MockPermissionsClient, context.CancelFunc)
		vnetMocks       func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr         string
	}{
		{
			name:   "fail: failed to get vnet",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, errors.New("failed to get vnet"))
			},
			wantErr: "failed to get vnet",
		},
		{
			name:   "fail: master subnet doesn't exist",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name:   "fail: worker subnet ID doesn't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].ID = to.StringPtr("not valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + workerSubnet + "' could not be found.",
		},
		{
			name:   "fail: permissions don't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			permissionMocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel context.CancelFunc) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, "Microsoft.Network", "", "natGateways", gomock.Any()).
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
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal does not have Network Contributor role on nat gateway '" + workerNgID + "'.",
		},
		{
			name:   "pass",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			permissionMocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel context.CancelFunc) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, "Microsoft.Network", "", "natGateways", gomock.Any()).
					AnyTimes().
					Return([]mgmtauthorization.Permission{
						{
							Actions: &[]string{
								"Microsoft.Network/natGateways/join/action",
								"Microsoft.Network/natGateways/read",
								"Microsoft.Network/natGateways/write",
							},
							NotActions: &[]string{},
						},
					}, nil)
			},
		},
		{
			name:   "pass: no nat gateway to check",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NatGateway = nil
				(*vnet.Subnets)[1].NatGateway = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			permissionsClient := mock_authorization.NewMockPermissionsClient(controller)
			vnetClient := mock_network.NewMockVirtualNetworksClient(controller)

			vnet := &mgmtnetwork.VirtualNetwork{
				ID: &vnetID,
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								NatGateway: &mgmtnetwork.SubResource{
									ID: &masterNgID,
								},
							},
						},
						{
							ID: &workerSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								NatGateway: &mgmtnetwork.SubResource{
									ID: &workerNgID,
								},
							},
						},
					},
				},
			}

			dv := &dynamic{
				authorizerType:  AuthorizerClusterServicePrincipal,
				log:             logrus.NewEntry(logrus.StandardLogger()),
				permissions:     permissionsClient,
				virtualNetworks: vnetClient,
			}

			if tt.permissionMocks != nil {
				tt.permissionMocks(permissionsClient, cancel)
			}

			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, *vnet)
			}

			err := dv.validateNatGatewayPermissions(ctx, tt.subnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateCIDRRanges(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name      string
		modifyOC  func(*api.OpenShiftCluster)
		vnetMocks func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr   string
	}{
		{
			name: "pass",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
		{
			name: "fail: conflicting ranges",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "10.0.0.0/24"
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided CIDRs must not overlap: '10.0.0.0/24 overlaps with 10.0.0.0/24'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

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
						{
							SubnetID: workerSubnet,
						},
					},
				},
			}

			vnets := []mgmtnetwork.VirtualNetwork{
				{
					ID:       &vnetID,
					Location: to.StringPtr("eastus"),
					Name:     to.StringPtr("VNET With AddressPrefix"),
					VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
						Subnets: &[]mgmtnetwork.Subnet{
							{
								ID: &masterSubnet,
								SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
									AddressPrefix: to.StringPtr("10.0.0.0/24"),
									NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
										ID: &masterNSGv1,
									},
								},
							},
							{
								ID: &workerSubnet,
								SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
									AddressPrefix: to.StringPtr("10.0.1.0/24"),
									NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
										ID: &workerNSGv1,
									},
								},
							},
						},
					},
				},
				{
					ID:       &vnetID,
					Location: to.StringPtr("eastus"),
					Name:     to.StringPtr("VNET With AddressPrefixes"),
					VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
						Subnets: &[]mgmtnetwork.Subnet{
							{
								ID: &masterSubnet,
								SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
									AddressPrefixes: to.StringSlicePtr([]string{"10.0.0.0/24"}),
									NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
										ID: &masterNSGv1,
									},
								},
							},
							{
								ID: &workerSubnet,
								SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
									AddressPrefixes: to.StringSlicePtr([]string{"10.0.1.0/24"}),
									NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
										ID: &workerNSGv1,
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

			for _, vnet := range vnets {
				vnetClient := mock_network.NewMockVirtualNetworksClient(controller)
				if tt.vnetMocks != nil {
					tt.vnetMocks(vnetClient, vnet)
				}

				dv := &dynamic{
					log:             logrus.NewEntry(logrus.StandardLogger()),
					virtualNetworks: vnetClient,
				}

				err := dv.validateCIDRRanges(ctx, []Subnet{
					{ID: masterSubnet},
					{ID: workerSubnet},
				},
					oc.Properties.NetworkProfile.PodCIDR, oc.Properties.NetworkProfile.ServiceCIDR)
				utilerror.AssertErrorMessage(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateVnetLocation(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name     string
		location string
		wantErr  string
	}{
		{
			name:     "pass",
			location: "eastus",
		},
		{
			name:     "fail: location differs",
			location: "neverland",
			wantErr:  "400: InvalidLinkedVNet: : The vnet location 'neverland' must match the cluster location 'eastus'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vnet := mgmtnetwork.VirtualNetwork{
				ID:       to.StringPtr(vnetID),
				Location: to.StringPtr(tt.location),
			}

			vnetClient := mock_network.NewMockVirtualNetworksClient(controller)
			vnetClient.EXPECT().
				Get(gomock.Any(), resourceGroupName, vnetName, "").
				Return(vnet, nil)

			dv := &dynamic{
				log:             logrus.NewEntry(logrus.StandardLogger()),
				virtualNetworks: vnetClient,
			}

			vnetr, err := azure.ParseResourceID(vnetID)
			if err != nil {
				t.Fatal(err)
			}

			err = dv.validateVnetLocation(ctx, vnetr, "eastus")
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateSubnets(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name      string
		modifyOC  func(*api.OpenShiftCluster)
		vnetMocks func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr   string
	}{
		{
			name: "pass",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
		{
			name: "pass (master subnet)",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.MasterProfile = api.MasterProfile{
					SubnetID: masterSubnet,
				}
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
		{
			name: "pass (cluster in creating provisioning status)",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
		{
			name: "fail: subnet does not exist on vnet",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name: "pass: subnet provisioning state is succeeded",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].ProvisioningState = mgmtnetwork.Succeeded
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
		{
			name: "fail: subnet provisioning state is not succeeded",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].ProvisioningState = mgmtnetwork.Failed
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '" + masterSubnet + "' is not in a Succeeded state",
		},
		{
			name: "pass: provisioning state creating: subnet has expected NSG attached",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
		{
			name: "fail: provisioning state creating: subnet has incorrect NSG attached",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup.ID = to.StringPtr("not-the-correct-nsg")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '" + masterSubnet + "' is invalid: must not have a network security group attached.",
		},
		{
			name: "fail: invalid architecture version returns no NSG",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ArchitectureVersion = 9001
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "unknown architecture version 9001",
		},
		{
			name: "fail: nsg id doesn't match expected",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup.ID = to.StringPtr("not matching")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '" + masterSubnet + "' is invalid: must have network security group '" + masterNSGv1 + "' attached.",
		},
		{
			name: "fail: invalid subnet CIDR",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].AddressPrefix = to.StringPtr("not-valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "invalid CIDR address: not-valid",
		},
		{
			name: "fail: too small subnet CIDR",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].AddressPrefix = to.StringPtr("10.0.0.0/28")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '" + masterSubnet + "' is invalid: must be /27 or larger.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
				},
			}
			vnet := mgmtnetwork.VirtualNetwork{
				ID: &vnetID,
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								AddressPrefix: to.StringPtr("10.0.0.0/24"),
								NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
									ID: &masterNSGv1,
								},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
						},
					},
				},
			}

			if tt.modifyOC != nil {
				tt.modifyOC(oc)
			}
			vnetClient := mock_network.NewMockVirtualNetworksClient(controller)
			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, vnet)
			}
			dv := &dynamic{
				virtualNetworks: vnetClient,
				log:             logrus.NewEntry(logrus.StandardLogger()),
			}

			err := dv.ValidateSubnets(ctx, oc, []Subnet{{ID: masterSubnet, Path: masterSubnetPath}})
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

// Unit tests for validateAccess using CheckAccessV2
// This will totally replace the current unit tests using ListPermissions (ListForResource)
// once fully migrated to CheckAccessV2

var mockAccessToken = azcore.AccessToken{
	Token:     "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiOjE2ODExNDk2NjksImV4cCI6MTcxMjY4NTY2OSwiYXVkIjoid3d3LmV4YW1wbGUuY29tIiwic3ViIjoianJvY2tldEBleGFtcGxlLmNvbSIsIkdpdmVuTmFtZSI6IkpvaG5ueSIsIlN1cm5hbWUiOiJSb2NrZXQiLCJFbWFpbCI6Impyb2NrZXRAZXhhbXBsZS5jb20iLCJSb2xlIjpbIk1hbmFnZXIiLCJQcm9qZWN0IEFkbWluaXN0cmF0b3IiXSwib2lkIjoiYmlsbHlxZWlsbG9yIn0.3MZk1YRSME8FW0l2DzXsIilEZh08CzjVopy30lbvLqQ", // mocked up data
	ExpiresOn: time.Now(),
}

func mockTokenCredential(tokenCred *mock_azcore.MockTokenCredential) {
	tokenCred.EXPECT().
		GetToken(gomock.Any(), gomock.Any()).
		Return(mockAccessToken, nil)
}

func TestValidateVnetPermissionsWithCheckAccess(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_azcore.MockTokenCredential, *mock_remotepdp.MockRemotePDPClient, context.CancelFunc)
		wantErr string
	}{
		{
			name: "pass",
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, _ context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&remotepdp.AuthorizationDecisionResponse{
						Value: []remotepdp.AuthorizationDecision{
							{
								AccessDecision: remotepdp.Allowed,
							}, {
								AccessDecision: remotepdp.Allowed,
							},
						},
					}, nil)
			},
		},
		{
			name: "fail: missing permissions",
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&remotepdp.AuthorizationDecisionResponse{
						Value: []remotepdp.AuthorizationDecision{
							{
								AccessDecision: remotepdp.NotAllowed,
							}, {
								AccessDecision: remotepdp.Allowed,
							},
						},
					}, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have Network Contributor role on vnet '" + vnetID + "'.",
		},
		{
			name: "fail: getting an invalid token from AAD",
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, _ *mock_remotepdp.MockRemotePDPClient, _ context.CancelFunc) {
				tokenCred.EXPECT().GetToken(gomock.Any(), gomock.Any()).
					Return(azcore.AccessToken{}, nil)
			},
			wantErr: "token contains an invalid number of segments",
		},
		{
			name: "fail: getting an error when calling CheckAccessV2",
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(nil, errors.New("Unexpected failure calling CheckAccessV2"))
			},
			wantErr: "Unexpected failure calling CheckAccessV2",
		},
		{
			name: "fail: getting a nil response from CheckAccessV2",
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(nil, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have Network Contributor role on vnet '" + vnetID + "'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			tokenCred := mock_azcore.NewMockTokenCredential(controller)

			pdpClient := mock_remotepdp.NewMockRemotePDPClient(controller)
			tt.mocks(tokenCred, pdpClient, cancel)

			dv := &dynamic{
				azEnv:                      &azureclient.PublicCloud,
				appID:                      "fff51942-b1f9-4119-9453-aaa922259eb7",
				authorizerType:             AuthorizerClusterServicePrincipal,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
				pdpClient:                  pdpClient,
				checkAccessSubjectInfoCred: tokenCred,
			}

			vnetr, err := azure.ParseResourceID(vnetID)
			if err != nil {
				t.Fatal(err)
			}

			err = dv.validateVnetPermissions(ctx, vnetr)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateRouteTablesPermissionsWithCheckAccess(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name           string
		subnet         Subnet
		pdpClientMocks func(*mock_azcore.MockTokenCredential, *mock_remotepdp.MockRemotePDPClient, context.CancelFunc)
		vnetMocks      func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr        string
	}{
		{
			name:   "fail: failed to get vnet",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, errors.New("failed to get vnet"))
			},
			wantErr: "failed to get vnet",
		},
		{
			name:   "fail: master subnet doesn't exist",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name:   "fail: worker subnet ID doesn't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].ID = to.StringPtr("not valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + workerSubnet + "' could not be found.",
		},
		{
			name:   "pass: no route table to check",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].RouteTable = nil
				(*vnet.Subnets)[1].RouteTable = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
		{
			name:   "fail: permissions don't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			pdpClientMocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&remotepdp.AuthorizationDecisionResponse{
						Value: []remotepdp.AuthorizationDecision{
							{
								AccessDecision: remotepdp.Allowed,
							}, {
								AccessDecision: remotepdp.NotAllowed,
							},
						},
					}, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal does not have Network Contributor role on route table '" + workerRtID + "'.",
		},
		{
			name:   "pass",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			pdpClientMocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&remotepdp.AuthorizationDecisionResponse{
						Value: []remotepdp.AuthorizationDecision{
							{
								AccessDecision: remotepdp.Allowed,
							}, {
								AccessDecision: remotepdp.Allowed,
							},
						},
					}, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			tokenCred := mock_azcore.NewMockTokenCredential(controller)

			pdpClient := mock_remotepdp.NewMockRemotePDPClient(controller)

			vnetClient := mock_network.NewMockVirtualNetworksClient(controller)

			vnet := &mgmtnetwork.VirtualNetwork{
				ID: &vnetID,
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								RouteTable: &mgmtnetwork.RouteTable{
									ID: &masterRtID,
								},
							},
						},
						{
							ID: &workerSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								RouteTable: &mgmtnetwork.RouteTable{
									ID: &workerRtID,
								},
							},
						},
					},
				},
			}

			dv := &dynamic{
				azEnv:                      &azureclient.PublicCloud,
				authorizerType:             AuthorizerClusterServicePrincipal,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
				checkAccessSubjectInfoCred: tokenCred,
				pdpClient:                  pdpClient,
				virtualNetworks:            vnetClient,
			}

			if tt.pdpClientMocks != nil {
				tt.pdpClientMocks(tokenCred, pdpClient, cancel)
			}

			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, *vnet)
			}

			err := dv.validateRouteTablePermissions(ctx, tt.subnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateNatGatewaysPermissionsWithCheckAccess(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name           string
		subnet         Subnet
		pdpClientMocks func(*mock_azcore.MockTokenCredential, *mock_remotepdp.MockRemotePDPClient, context.CancelFunc)
		vnetMocks      func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr        string
	}{
		{
			name:   "fail: failed to get vnet",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, errors.New("failed to get vnet"))
			},
			wantErr: "failed to get vnet",
		},
		{
			name:   "fail: master subnet doesn't exist",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name:   "fail: worker subnet ID doesn't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].ID = to.StringPtr("not valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + workerSubnet + "' could not be found.",
		},
		{
			name:   "fail: permissions don't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			pdpClientMocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.
					EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&remotepdp.AuthorizationDecisionResponse{
						Value: []remotepdp.AuthorizationDecision{
							{
								AccessDecision: remotepdp.Allowed,
							}, {
								AccessDecision: remotepdp.NotAllowed,
							},
						},
					}, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal does not have Network Contributor role on nat gateway '" + workerNgID + "'.",
		},
		{
			name:   "pass",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			pdpClientMocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&remotepdp.AuthorizationDecisionResponse{
						Value: []remotepdp.AuthorizationDecision{
							{
								AccessDecision: remotepdp.Allowed,
							}, {
								AccessDecision: remotepdp.Allowed,
							},
						},
					}, nil)
			},
		},
		{
			name:   "pass: no nat gateway to check",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NatGateway = nil
				(*vnet.Subnets)[1].NatGateway = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			vnetClient := mock_network.NewMockVirtualNetworksClient(controller)

			tokenCred := mock_azcore.NewMockTokenCredential(controller)

			pdpClient := mock_remotepdp.NewMockRemotePDPClient(controller)

			vnet := &mgmtnetwork.VirtualNetwork{
				ID: &vnetID,
				VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
					Subnets: &[]mgmtnetwork.Subnet{
						{
							ID: &masterSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								NatGateway: &mgmtnetwork.SubResource{
									ID: &masterNgID,
								},
							},
						},
						{
							ID: &workerSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								NatGateway: &mgmtnetwork.SubResource{
									ID: &workerNgID,
								},
							},
						},
					},
				},
			}

			dv := &dynamic{
				azEnv:                      &azureclient.PublicCloud,
				authorizerType:             AuthorizerClusterServicePrincipal,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
				checkAccessSubjectInfoCred: tokenCred,
				pdpClient:                  pdpClient,
				virtualNetworks:            vnetClient,
			}

			if tt.pdpClientMocks != nil {
				tt.pdpClientMocks(tokenCred, pdpClient, cancel)
			}

			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, *vnet)
			}

			err := dv.validateNatGatewayPermissions(ctx, tt.subnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestCheckBYONsg(t *testing.T) {
	subnetWithNSG := &mgmtnetwork.Subnet{
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
			NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
				ID: &masterNSGv1,
			},
		},
	}
	subnetWithoutNSG := &mgmtnetwork.Subnet{
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{},
	}

	for _, tt := range []struct {
		name       string
		subnetByID map[string]*mgmtnetwork.Subnet
		byoNSG     bool
		wantErr    string
	}{
		{
			name: "pass: all subnets are attached (BYONSG)",
			subnetByID: map[string]*mgmtnetwork.Subnet{
				"A": subnetWithNSG,
				"B": subnetWithNSG,
			},
			byoNSG: true,
		},
		{
			name: "pass: no subnets are attached (no longer BYONSG)",
			subnetByID: map[string]*mgmtnetwork.Subnet{
				"A": subnetWithoutNSG,
				"B": subnetWithoutNSG,
			},
			byoNSG: false,
		},
		{
			name: "fail: parts of the subnets are attached",
			subnetByID: map[string]*mgmtnetwork.Subnet{
				"A": subnetWithNSG,
				"B": subnetWithoutNSG,
				"C": subnetWithNSG,
			},
			byoNSG:  false,
			wantErr: "400: InvalidLinkedVNet: : When the enable-preconfigured-nsg option is specified, both the master and worker subnets should have network security groups (NSG) attached to them before starting the cluster installation.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dv := &dynamic{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}
			byoNSG, err := dv.checkByoNSG(tt.subnetByID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if byoNSG != tt.byoNSG {
				t.Errorf("byoNSG got %t, want %t", byoNSG, tt.byoNSG)
			}
		})
	}
}
