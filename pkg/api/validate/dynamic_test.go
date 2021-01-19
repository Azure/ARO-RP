package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

func TestValidateVnetPermissions(t *testing.T) {
	ctx := context.Background()

	resourceGroupName := "testGroup"
	vnetName := "testVnet"
	subscriptionID := "0000000-0000-0000-0000-000000000000"
	vnetID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroupName + "/providers/Microsoft.Network/virtualNetworks/" + vnetName
	resourceType := "virtualNetworks"
	resourceProvider := "Microsoft.Network"

	controller := gomock.NewController(t)
	defer controller.Finish()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_authorization.MockPermissionsClient, func())
		wantErr string
	}{
		{
			name: "pass",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
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
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
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
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider does not have Network Contributor permission on vnet '" + vnetID + "'.",
		},
		{
			name: "fail: not found",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			permissionsClient := mock_authorization.NewMockPermissionsClient(controller)
			tt.mocks(permissionsClient, cancel)

			dv := &dynamic{
				log:         logrus.NewEntry(logrus.StandardLogger()),
				permissions: permissionsClient,
				vnetr: &azure.Resource{
					ResourceGroup:  resourceGroupName,
					ResourceType:   resourceType,
					Provider:       resourceProvider,
					ResourceName:   vnetName,
					SubscriptionID: subscriptionID,
				},
			}

			err := dv.ValidateVnetPermissions(ctx, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestGetRouteTableID(t *testing.T) {
	resourceGroupID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup"
	vnetID := resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/testVnet"
	genericSubnet := vnetID + "/subnet/genericSubnet"
	routeTableID := resourceGroupID + "/providers/Microsoft.Network/routeTables/testRouteTable"

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
			wantErr: "400: InvalidLinkedVNet: : The subnet '" + genericSubnet + "' could not be found.",
		},
	} {
		vnet := &mgmtnetwork.VirtualNetwork{
			ID: &vnetID,
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				Subnets: &[]mgmtnetwork.Subnet{
					{
						ID: &genericSubnet,
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							RouteTable: &mgmtnetwork.RouteTable{
								ID: &routeTableID,
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
		_, err := getRouteTableID(vnet, "", genericSubnet)
		if err != nil && err.Error() != tt.wantErr ||
			err == nil && tt.wantErr != "" {
			t.Error(err)
		}
	}
}

func TestValidateVnetDNS(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	resourceGroupName := "testGroup"
	vnetName := "testVnet"
	vnetID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroupName + "/providers/Microsoft.Network/virtualNetworks/" + vnetName

	for _, tt := range []struct {
		name      string
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
			name: "fail: dhcp options set",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnet.DhcpOptions = &mgmtnetwork.DhcpOptions{
					DNSServers: &[]string{
						"8.8.8.8",
					},
				}
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided vnet '" + vnetID + "' is invalid: custom DNS servers are not supported.",
		},
		{
			name: "fail: failed to get vnet",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, errors.New("failed to get vnet"))
			},
			wantErr: "failed to get vnet",
		},
	} {
		vnet := mgmtnetwork.VirtualNetwork{
			ID: to.StringPtr(vnetID),
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				DhcpOptions: nil,
			},
		}

		vnetClient := mock_network.NewMockVirtualNetworksClient(controller)
		tt.vnetMocks(vnetClient, vnet)

		dv := &dynamic{
			log:             logrus.NewEntry(logrus.StandardLogger()),
			virtualNetworks: vnetClient,
			vnetr: &azure.Resource{
				ResourceGroup: resourceGroupName,
				ResourceName:  vnetName,
			},
		}

		err := dv.ValidateVnetDNS(ctx)
		if err != nil && err.Error() != tt.wantErr ||
			err == nil && tt.wantErr != "" {
			t.Error(err)
		}
	}
}

func TestValidateRouteTablePermissions(t *testing.T) {
	ctx := context.Background()

	resourceGroupName := "testGroup"
	resourceGroupID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroupName
	routeTableName := "testRouteTable"
	routeTableID := resourceGroupID + "/providers/Microsoft.Network/routeTables/" + routeTableName

	controller := gomock.NewController(t)
	defer controller.Finish()

	for _, tt := range []struct {
		name    string
		rtID    string
		mocks   func(*mock_authorization.MockPermissionsClient, func())
		wantErr string
	}{
		{
			name: "pass",
			rtID: routeTableID,
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, "Microsoft.Network", "", "routeTables", routeTableName).
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
			name:    "fail: cannot parse resource id",
			rtID:    "invalid_route_table_id",
			wantErr: "parsing failed for invalid_route_table_id. Invalid resource Id format",
		},
		{
			name: "fail: missing permissions",
			rtID: routeTableID,
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, "Microsoft.Network", "", "routeTables", routeTableName).
					Do(func(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) {
						cancel()
					}).
					Return([]mgmtauthorization.Permission{
						{
							Actions:    &[]string{},
							NotActions: &[]string{},
						},
					}, nil)
			},
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider does not have Network Contributor permission on route table '" + routeTableID + "'.",
		},
		{
			name: "fail: not found",
			rtID: routeTableID,
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), resourceGroupName, "Microsoft.Network", "", "routeTables", routeTableName).
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
			wantErr: "400: InvalidLinkedRouteTable: : The route table '" + routeTableID + "' could not be found.",
		},
	} {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		permissionsClient := mock_authorization.NewMockPermissionsClient(controller)
		if tt.mocks != nil {
			tt.mocks(permissionsClient, cancel)
		}

		dv := &dynamic{
			log:         logrus.NewEntry(logrus.StandardLogger()),
			permissions: permissionsClient,
		}

		// purposefully hardcoding path to "" so it is not needed in the wantErr message
		err := dv.validateRouteTablePermissions(ctx, tt.rtID, "", api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
		if err != nil && err.Error() != tt.wantErr ||
			err == nil && tt.wantErr != "" {
			t.Error(err)
		}
	}
}

func TestValidateRouteTablesPermissions(t *testing.T) {
	ctx := context.Background()

	subscriptionID := "0000000-0000-0000-0000-000000000000"
	resourceGroupName := "testGroup"
	resourceGroupID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroupName
	vnetName := "testVnet"
	vnetID := resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/" + vnetName
	masterSubnet := vnetID + "/subnet/masterSubnet"
	workerSubnet := vnetID + "/subnet/workerSubnet"
	masterRtID := resourceGroupID + "/providers/Microsoft.Network/routeTables/masterRt"
	workerRtID := resourceGroupID + "/providers/Microsoft.Network/routeTables/workerRt"

	controller := gomock.NewController(t)
	defer controller.Finish()

	for _, tt := range []struct {
		name            string
		permissionMocks func(*mock_authorization.MockPermissionsClient, func())
		vnetMocks       func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr         string
	}{
		{
			name: "fail: failed to get vnet",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, errors.New("failed to get vnet"))
			},
			wantErr: "failed to get vnet",
		},
		{
			name: "fail: master subnet doesn't exist",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name: "fail: worker subnet ID doesn't exist",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].ID = to.StringPtr("not valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: properties.workerProfiles[0].subnetId: The subnet '" + workerSubnet + "' could not be found.",
		},
		{
			name: "fail: permissions don't exist",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			permissionMocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), strings.ToLower(resourceGroupName), strings.ToLower("Microsoft.Network"), "", strings.ToLower("routeTables"), gomock.Any()).
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
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider does not have Network Contributor permission on route table '" + strings.ToLower(masterRtID) + "'.",
		},
		{
			name: "pass",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			permissionMocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), strings.ToLower(resourceGroupName), strings.ToLower("Microsoft.Network"), "", strings.ToLower("routeTables"), gomock.Any()).
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
	} {
		t.Run(tt.name, func(t *testing.T) {
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
				log:             logrus.NewEntry(logrus.StandardLogger()),
				permissions:     permissionsClient,
				virtualNetworks: vnetClient,

				vnetr: &azure.Resource{
					ResourceGroup:  resourceGroupName,
					ResourceName:   vnetName,
					SubscriptionID: subscriptionID,
					Provider:       "Microsoft.Network",
					ResourceType:   "virtualNetworks",
				},

				masterSubnetID:  masterSubnet,
				workerSubnetIDs: []string{workerSubnet},
			}

			if tt.permissionMocks != nil {
				tt.permissionMocks(permissionsClient, cancel)
			}

			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, *vnet)
			}

			err := dv.ValidateRouteTablesPermissions(ctx, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
