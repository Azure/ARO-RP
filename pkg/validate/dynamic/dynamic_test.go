package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	mock_remotepdp "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/authz/remotepdp"
	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
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
	workerSubnetPath  = "properties.workerProfile.subnetId"
	masterRtID        = resourceGroupID + "/providers/Microsoft.Network/routeTables/masterRt"
	workerRtID        = resourceGroupID + "/providers/Microsoft.Network/routeTables/workerRt"
	masterNgID        = resourceGroupID + "/providers/Microsoft.Network/natGateways/masterNg"
	workerNgID        = resourceGroupID + "/providers/Microsoft.Network/natGateways/workerNg"
	masterNSGv1       = resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg"
	workerNSGv1       = resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg"
	platformIdentity1 = resourceGroupID + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/miwi-platform-identity"
	dummyClientId     = uuid.DefaultGenerator.Generate()
	dummyObjectId     = uuid.DefaultGenerator.Generate()
)

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
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGDisabled
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
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGDisabled
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
			name: "pass: byonsg skips validating if an NSG is attached",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup.ID = to.StringPtr("attached")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
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
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateUpdating
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGDisabled
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '" + masterSubnet + "' is invalid: must have network security group '" + masterNSGv1 + "' attached.",
		},
		{
			name: "pass: byonsg doesn't check if nsg ids are matched after creating",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup.ID = to.StringPtr("don't care what it is")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
				oc.Properties.ProvisioningState = api.ProvisioningStateUpdating
			},
		},
		{
			name: "fail: no nsg attached during update",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup.ID = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateUpdating
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGDisabled
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '" + masterSubnet + "' is invalid: must have network security group '" + masterNSGv1 + "' attached.",
		},
		{
			name: "fail: byonsg requires an nsg to be attached during update",
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup.ID = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateUpdating
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			wantErr: "400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '" + masterSubnet + "' is invalid: must have a network security group attached.",
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

var (
	platformIdentity1SubnetActions = []string{
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/routeTables/write",
		"Microsoft.Network/natGateways/read",
		"Microsoft.Network/networkSecurityGroups/join/action",
		"Microsoft.Compute/diskEncryptionSets/read",
	}
	platformIdentity1SubnetActionsNoIntersect = []string{
		"Microsoft.Network/virtualNetworks/nointersect/nointersect",
	}
	platformIdentities = map[string]api.PlatformWorkloadIdentity{
		"Dummy": {
			ResourceID: platformIdentity1,
			ClientID:   dummyClientId,
			ObjectID:   dummyObjectId,
		},
	}
	validSubnetsAuthorizationDecisions = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/read",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/write",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/read",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/write",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}

	invalidSubnetsAuthorizationDecisionsReadNotAllowed = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/read",
				AccessDecision: remotepdp.NotAllowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/write",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/read",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/write",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}

	invalidSubnetsAuthorizationDecisionsMissingWrite = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/read",
				AccessDecision: remotepdp.NotAllowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/write",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/read",
				AccessDecision: remotepdp.Allowed,
			},
			// deliberately missing subnets write
		},
	}
)

func mockTokenCredential(tokenCred *mock_azcore.MockTokenCredential) {
	tokenCred.EXPECT().
		GetToken(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(mockAccessToken, nil)
}

func TestValidateVnetPermissions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                string
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		mocks               func(*mock_azcore.MockTokenCredential, *mock_remotepdp.MockRemotePDPClient, context.CancelFunc)
		wantErr             string
	}{
		{
			name: "pass",
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, _ context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validSubnetsAuthorizationDecisions, nil)
			},
		},
		{
			name:               "pass - MIWI Cluster",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, _ context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validSubnetsAuthorizationDecisions, nil)
			},
		},
		{
			name:               "Success - MIWI Cluster - No intersecting Subnet Actions",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActionsNoIntersect,
			},
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, _ context.CancelFunc) {
				mockTokenCredential(tokenCred)
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
					Return(&invalidSubnetsAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have Network Contributor role on vnet '" + vnetID + "'.",
		},
		{
			name:               "Fail - MIWI Cluster - missing permissions",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidSubnetsAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on vnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet'.",
		},
		{
			name: "fail: CheckAccess Return less entries than requested",
			mocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidSubnetsAuthorizationDecisionsMissingWrite, nil)
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
				appID:                      to.StringPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
				authorizerType:             AuthorizerClusterServicePrincipal,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
				pdpClient:                  pdpClient,
				checkAccessSubjectInfoCred: tokenCred,
			}

			if tt.platformIdentities != nil {
				dv.platformIdentities = tt.platformIdentities
				dv.platformIdentitiesActionsMap = tt.platformIdentityMap
				dv.authorizerType = AuthorizerWorkloadIdentity
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

var (
	invalidRouteTablesAuthorizationDecisionsWriteNotAllowed = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/routeTables/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/read",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/write",
				AccessDecision: remotepdp.NotAllowed,
			},
		},
	}
	invalidRouteTablesAuthorizationDecisionsMissingWrite = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/routeTables/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/read",
				AccessDecision: remotepdp.Allowed,
			},
			// deliberately missing routetables write
		},
	}
	validRouteTablesAuthorizationDecisions = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/routeTables/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/read",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/write",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}
)

func TestValidateRouteTablesPermissions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                string
		subnet              Subnet
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		pdpClientMocks      func(*mock_azcore.MockTokenCredential, *mock_remotepdp.MockRemotePDPClient, context.CancelFunc)
		vnetMocks           func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr             string
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
					Return(&invalidRouteTablesAuthorizationDecisionsWriteNotAllowed, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal does not have Network Contributor role on route table '" + workerRtID + "'.",
		},
		{
			name:               "Fail - MIWI Cluster - permissions don't exist",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
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
					Return(&invalidRouteTablesAuthorizationDecisionsWriteNotAllowed, nil)
			},
			wantErr: "400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on route table '" + workerRtID + "'.",
		},
		{
			name:   "fail: CheckAccessV2 doesn't return all the entries",
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
					Return(&invalidRouteTablesAuthorizationDecisionsMissingWrite, nil)
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
					Return(&validRouteTablesAuthorizationDecisions, nil)
			},
		},
		{
			name:               "pass - MIWI Cluster",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			pdpClientMocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validRouteTablesAuthorizationDecisions, nil)
			},
		},
		{
			name:               "Success - MIWI Cluster - No intersecting Subnet Actions",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActionsNoIntersect,
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			pdpClientMocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
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
				appID:                      to.StringPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
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

			if tt.platformIdentities != nil {
				dv.platformIdentities = tt.platformIdentities
				dv.platformIdentitiesActionsMap = tt.platformIdentityMap
				dv.authorizerType = AuthorizerWorkloadIdentity
			}

			err := dv.validateRouteTablePermissions(ctx, tt.subnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

var (
	invalidNatGWAuthorizationDecisionsReadNotAllowed = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/natGateways/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/read",
				AccessDecision: remotepdp.NotAllowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/write",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}
	invalidNatGWAuthorizationDecisionsMissingWrite = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/natGateways/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/read",
				AccessDecision: remotepdp.Allowed,
			},
			// deliberately missing natGateway write
		},
	}
	validNatGWAuthorizationDecision = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/natGateways/join/action",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/read",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/write",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}
)

func TestValidateNatGatewaysPermissions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                string
		subnet              Subnet
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		pdpClientMocks      func(*mock_azcore.MockTokenCredential, *mock_remotepdp.MockRemotePDPClient, context.CancelFunc)
		vnetMocks           func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr             string
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
					Return(&invalidNatGWAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal does not have Network Contributor role on nat gateway '" + workerNgID + "'.",
		},
		{
			name:               "Fail - MIWI Cluster - permissions don't exist",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
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
					Return(&invalidNatGWAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on nat gateway '" + workerNgID + "'.",
		},
		{
			name:   "fail: CheckAccessV2 doesn't return all permissions",
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
					Return(&invalidNatGWAuthorizationDecisionsMissingWrite, nil)
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
					Return(&validNatGWAuthorizationDecision, nil)
			},
		},
		{
			name:               "pass - MIWI Cluster",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			pdpClientMocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validNatGWAuthorizationDecision, nil)
			},
		},
		{
			name:               "Success - MIWI Cluster - No intersecting Subnet Actions",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActionsNoIntersect,
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					Return(vnet, nil)
			},
			pdpClientMocks: func(tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_remotepdp.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
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
				appID:                      to.StringPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
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

			if tt.platformIdentities != nil {
				dv.platformIdentities = tt.platformIdentities
				dv.platformIdentitiesActionsMap = tt.platformIdentityMap
				dv.authorizerType = AuthorizerWorkloadIdentity
			}

			err := dv.validateNatGatewayPermissions(ctx, tt.subnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestCheckPreconfiguredNSG(t *testing.T) {
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
		wantErr    string
	}{
		{
			name: "pass: all subnets are attached",
			subnetByID: map[string]*mgmtnetwork.Subnet{
				"A": subnetWithNSG,
				"B": subnetWithNSG,
			},
		},
		{
			name: "fail: no subnets are attached",
			subnetByID: map[string]*mgmtnetwork.Subnet{
				"A": subnetWithoutNSG,
				"B": subnetWithoutNSG,
			},
			wantErr: "400: InvalidLinkedVNet: : When the enable-preconfigured-nsg option is specified, both the master and worker subnets should have network security groups (NSG) attached to them before starting the cluster installation.",
		},
		{
			name: "fail: parts of the subnets are attached",
			subnetByID: map[string]*mgmtnetwork.Subnet{
				"A": subnetWithNSG,
				"B": subnetWithoutNSG,
				"C": subnetWithNSG,
			},
			wantErr: "400: InvalidLinkedVNet: : When the enable-preconfigured-nsg option is specified, both the master and worker subnets should have network security groups (NSG) attached to them before starting the cluster installation.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dv := &dynamic{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}
			err := dv.checkPreconfiguredNSG(tt.subnetByID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

var (
	canJoinNSG = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/networkSecurityGroups/join/action",
				AccessDecision: remotepdp.Allowed},
		},
	}

	cannotJoinNSG = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/networkSecurityGroups/join/action",
				AccessDecision: remotepdp.NotAllowed},
		},
	}
)

func TestValidatePreconfiguredNSGPermissions(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name                string
		modifyOC            func(*api.OpenShiftCluster)
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		checkAccessMocks    func(context.CancelFunc, *mock_remotepdp.MockRemotePDPClient, *mock_azcore.MockTokenCredential)
		vnetMocks           func(*mock_network.MockVirtualNetworksClient, mgmtnetwork.VirtualNetwork)
		wantErr             string
	}{
		{
			name: "pass: skip when preconfiguredNSG is not enabled",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGDisabled
			},
		},
		{
			name: "fail: sp doesn't have the permission on all NSGs",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, authReq remotepdp.AuthorizationRequest) (*remotepdp.AuthorizationDecisionResponse, error) {
						cancel() // wait.PollImmediateUntil will always be invoked at least once
						switch authReq.Resource.Id {
						case masterNSGv1:
							return &canJoinNSG, nil
						case workerNSGv1:
							return &cannotJoinNSG, nil
						}
						return &cannotJoinNSG, nil
					},
					).AnyTimes()
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have Network Contributor role on network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg'. This is required when the enable-preconfigured-nsg option is specified.",
		},
		{
			name: "Fail - MIWI Cluster - permissions don't exist on all nsg",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, authReq remotepdp.AuthorizationRequest) (*remotepdp.AuthorizationDecisionResponse, error) {
						cancel() // wait.PollImmediateUntil will always be invoked at least once
						switch authReq.Resource.Id {
						case masterNSGv1:
							return &canJoinNSG, nil
						case workerNSGv1:
							return &cannotJoinNSG, nil
						}
						return &cannotJoinNSG, nil
					},
					).AnyTimes()
			},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			wantErr: "400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg'. This is required when the enable-preconfigured-nsg option is specified.",
		},
		{
			name: "pass: sp has the required permission on the NSG",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(_, _ interface{}) {
						cancel()
					}).
					Return(&canJoinNSG, nil).
					AnyTimes()
			},
		},
		{
			name: "pass - MIWI Cluster",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(_, _ interface{}) {
						cancel()
					}).
					Return(&canJoinNSG, nil).
					AnyTimes()
			},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
		},
		{
			name: "Success - MIWI Cluster - No intersecting Subnet Actions",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_network.MockVirtualNetworksClient, vnet mgmtnetwork.VirtualNetwork) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, "").
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(_, _ interface{}) {
						cancel()
					}).
					Return(&canJoinNSG, nil).
					AnyTimes()
			},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActionsNoIntersect,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

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
						{
							ID: &workerSubnet,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								AddressPrefix: to.StringPtr("10.0.1.0/24"),
								NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
									ID: &workerNSGv1,
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

			tokenCred := mock_azcore.NewMockTokenCredential(controller)
			pdpClient := mock_remotepdp.NewMockRemotePDPClient(controller)

			if tt.checkAccessMocks != nil {
				tt.checkAccessMocks(cancel, pdpClient, tokenCred)
			}

			dv := &dynamic{
				azEnv:                      &azureclient.PublicCloud,
				appID:                      to.StringPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
				authorizerType:             AuthorizerClusterServicePrincipal,
				virtualNetworks:            vnetClient,
				pdpClient:                  pdpClient,
				checkAccessSubjectInfoCred: tokenCred,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
			}

			subnets := []Subnet{
				{ID: masterSubnet,
					Path: masterSubnetPath},
				{
					ID:   workerSubnet,
					Path: workerSubnetPath,
				},
			}

			if tt.platformIdentities != nil {
				dv.platformIdentities = tt.platformIdentities
				dv.platformIdentitiesActionsMap = tt.platformIdentityMap
				dv.authorizerType = AuthorizerWorkloadIdentity
			}

			err := dv.ValidatePreConfiguredNSGs(ctx, oc, subnets)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateSubnetSize(t *testing.T) {
	subnetId := "id"
	subnetPath := "path"
	for _, tt := range []struct {
		name    string
		address string
		subnet  Subnet
		wantErr string
	}{
		{
			name:    "subnet size is too small",
			address: "10.0.0.0/32",
			subnet:  Subnet{ID: subnetId, Path: subnetPath},
			wantErr: fmt.Sprintf("400: InvalidLinkedVNet: %s: The provided subnet '%s' is invalid: must be /27 or larger.", subnetPath, subnetId),
		},
		{
			name:    "subnet size is gucci gang",
			address: "10.0.0.0/27",
			subnet:  Subnet{ID: subnetId, Path: subnetPath},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubnetSize(tt.subnet, tt.address)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
