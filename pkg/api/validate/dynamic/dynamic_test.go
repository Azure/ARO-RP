package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
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
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal does not have Network Contributor permission on vnet '" + vnetID + "'.",
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			permissionsClient := mock_authorization.NewMockPermissionsClient(controller)
			tt.mocks(permissionsClient, cancel)

			dv := &dynamic{
				authorizerType: AuthorizerClusterServicePrincipal,
				log:            logrus.NewEntry(logrus.StandardLogger()),
				permissions:    permissionsClient,
			}

			vnetr, err := azure.ParseResourceID(vnetID)
			if err != nil {
				t.Fatal(err)
			}

			err = dv.validateVnetPermissions(ctx, vnetr)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(fmt.Errorf("\n%s\n !=\n%s", err.Error(), tt.wantErr))
			}
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
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(fmt.Errorf("\n%s\n !=\n%s", err.Error(), tt.wantErr))
			}
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
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal does not have Network Contributor permission on route table '" + workerRtID + "'.",
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
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(fmt.Errorf("\n%s\n !=\n%s", err.Error(), tt.wantErr))
			}
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
					{ID: workerSubnet}},
					oc.Properties.NetworkProfile.PodCIDR, oc.Properties.NetworkProfile.ServiceCIDR)
				if err != nil && err.Error() != tt.wantErr ||
					err == nil && tt.wantErr != "" {
					t.Error(*vnet.Name, err)
				}
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
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
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
			name: "fail: subnet doe not exist on vnet",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name: "fail: provisioning state creating: subnet has NSG",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
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
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(fmt.Errorf("\n%s\n !=\n%s", err.Error(), tt.wantErr))
			}
		})
	}
}
