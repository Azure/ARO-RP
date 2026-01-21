package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/checkaccess-v2-go-sdk/client"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	mock_checkaccess "github.com/Azure/ARO-RP/pkg/util/mocks/checkaccess"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	"github.com/Azure/ARO-RP/test/util/token"
)

var (
	resourceGroupName       = "testGroup"
	subscriptionID          = "0000000-0000-0000-0000-000000000000"
	resourceGroupID         = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroupName
	clusterName             = "cluster"
	clusterID               = resourceGroupID + "/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + clusterName
	vnetName                = "testVnet"
	vnetID                  = resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/" + vnetName
	masterSubnet            = vnetID + "/subnets/masterSubnet"
	workerSubnet            = vnetID + "/subnets/workerSubnet"
	masterSubnetPath        = "properties.masterProfile.subnetId"
	workerSubnetPath        = "properties.workerProfile.subnetId"
	masterRtID              = resourceGroupID + "/providers/Microsoft.Network/routeTables/masterRt"
	workerRtID              = resourceGroupID + "/providers/Microsoft.Network/routeTables/workerRt"
	masterNgID              = resourceGroupID + "/providers/Microsoft.Network/natGateways/masterNg"
	workerNgID              = resourceGroupID + "/providers/Microsoft.Network/natGateways/workerNg"
	masterNSGv1             = resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg"
	workerNSGv1             = resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg"
	platformIdentity1       = resourceGroupID + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/miwi-platform-identity"
	platformIdentity1SAName = "system:serviceaccount:dummy:dummy1"
	dummyClientId           = uuid.DefaultGenerator.Generate()
	dummyObjectId           = uuid.DefaultGenerator.Generate()
)

func TestGetRouteTableID(t *testing.T) {
	for _, tt := range []struct {
		name       string
		modifyVnet func(*sdknetwork.VirtualNetwork)
		wantErr    string
	}{
		{
			name: "pass",
		},
		{
			name: "pass: no route table",
			modifyVnet: func(vnet *sdknetwork.VirtualNetwork) {
				vnet.Properties.Subnets[0].Properties.RouteTable = nil
			},
		},
		{
			name: "fail: can't find subnet",
			modifyVnet: func(vnet *sdknetwork.VirtualNetwork) {
				vnet.Properties.Subnets = nil
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vnet := &sdknetwork.VirtualNetwork{
				ID: &vnetID,
				Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
					Subnets: []*sdknetwork.Subnet{
						{
							ID: &masterSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								RouteTable: &sdknetwork.RouteTable{
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
		modifyVnet func(*sdknetwork.VirtualNetwork)
		wantErr    string
	}{
		{
			name: "pass",
		},
		{
			name: "pass: no nat gateway",
			modifyVnet: func(vnet *sdknetwork.VirtualNetwork) {
				vnet.Properties.Subnets[0].Properties.NatGateway = nil
			},
		},
		{
			name: "fail: can't find subnet",
			modifyVnet: func(vnet *sdknetwork.VirtualNetwork) {
				vnet.Properties.Subnets = nil
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vnet := &sdknetwork.VirtualNetwork{
				ID: &vnetID,
				Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
					Subnets: []*sdknetwork.Subnet{
						{
							ID: &masterSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								NatGateway: &sdknetwork.SubResource{
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
		vnetMocks func(*mock_armnetwork.MockVirtualNetworksClient, sdknetwork.VirtualNetworksClientGetResponse)
		wantErr   string
	}{
		{
			name: "pass",
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
		{
			name: "fail: conflicting ranges",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "10.0.0.0/24"
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
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

			vnets := []sdknetwork.VirtualNetwork{
				{
					ID:       &vnetID,
					Location: pointerutils.ToPtr("eastus"),
					Name:     pointerutils.ToPtr("VNET With AddressPrefix"),
					Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
						Subnets: []*sdknetwork.Subnet{
							{
								ID: &masterSubnet,
								Properties: &sdknetwork.SubnetPropertiesFormat{
									AddressPrefix: pointerutils.ToPtr("10.0.0.0/24"),
									NetworkSecurityGroup: &sdknetwork.SecurityGroup{
										ID: &masterNSGv1,
									},
								},
							},
							{
								ID: &workerSubnet,
								Properties: &sdknetwork.SubnetPropertiesFormat{
									AddressPrefix: pointerutils.ToPtr("10.0.1.0/24"),
									NetworkSecurityGroup: &sdknetwork.SecurityGroup{
										ID: &workerNSGv1,
									},
								},
							},
						},
					},
				},
				{
					ID:       &vnetID,
					Location: pointerutils.ToPtr("eastus"),
					Name:     pointerutils.ToPtr("VNET With AddressPrefixes"),
					Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
						Subnets: []*sdknetwork.Subnet{
							{
								ID: &masterSubnet,
								Properties: &sdknetwork.SubnetPropertiesFormat{
									AddressPrefixes: []*string{pointerutils.ToPtr("10.0.0.0/24")},
									NetworkSecurityGroup: &sdknetwork.SecurityGroup{
										ID: &masterNSGv1,
									},
								},
							},
							{
								ID: &workerSubnet,
								Properties: &sdknetwork.SubnetPropertiesFormat{
									AddressPrefixes: []*string{pointerutils.ToPtr("10.0.1.0/24")},
									NetworkSecurityGroup: &sdknetwork.SecurityGroup{
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
				vnetClient := mock_armnetwork.NewMockVirtualNetworksClient(controller)
				if tt.vnetMocks != nil {
					tt.vnetMocks(vnetClient, sdknetwork.VirtualNetworksClientGetResponse{VirtualNetwork: vnet})
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

			vnet := sdknetwork.VirtualNetworksClientGetResponse{
				VirtualNetwork: sdknetwork.VirtualNetwork{
					ID:       pointerutils.ToPtr(vnetID),
					Location: pointerutils.ToPtr(tt.location),
				},
			}

			vnetClient := mock_armnetwork.NewMockVirtualNetworksClient(controller)
			vnetClient.EXPECT().
				Get(gomock.Any(), resourceGroupName, vnetName, nil).
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
		vnetMocks func(*mock_armnetwork.MockVirtualNetworksClient, sdknetwork.VirtualNetworksClientGetResponse)
		wantErr   string
	}{
		{
			name: "pass",
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
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
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
		{
			name: "pass (cluster in creating provisioning status)",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.NetworkSecurityGroup = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
		{
			name: "fail: subnet does not exist on vnet",
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name: "pass: subnet provisioning state is succeeded",
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.ProvisioningState = pointerutils.ToPtr(sdknetwork.ProvisioningStateSucceeded)
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
		{
			name: "fail: subnet provisioning state is not succeeded",
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.ProvisioningState = pointerutils.ToPtr(sdknetwork.ProvisioningStateFailed)
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
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
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
		{
			name: "fail: provisioning state creating: subnet has incorrect NSG attached",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGDisabled
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.NetworkSecurityGroup.ID = pointerutils.ToPtr("not-the-correct-nsg")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
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
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.NetworkSecurityGroup.ID = pointerutils.ToPtr("attached")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
		{
			name: "fail: invalid architecture version returns no NSG",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ArchitectureVersion = 9001
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			wantErr: "unknown architecture version 9001",
		},
		{
			name: "fail: nsg id doesn't match expected",
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.NetworkSecurityGroup.ID = pointerutils.ToPtr("not matching")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
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
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.NetworkSecurityGroup.ID = pointerutils.ToPtr("don't care what it is")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
				oc.Properties.ProvisioningState = api.ProvisioningStateUpdating
			},
		},
		{
			name: "fail: no nsg attached during update",
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.NetworkSecurityGroup.ID = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
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
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.NetworkSecurityGroup.ID = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
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
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.AddressPrefix = pointerutils.ToPtr("not-valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			wantErr: "invalid CIDR address: not-valid",
		},
		{
			name: "fail: too small subnet CIDR",
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.AddressPrefix = pointerutils.ToPtr("10.0.0.0/28")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
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
			vnet := sdknetwork.VirtualNetwork{
				ID: &vnetID,
				Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
					Subnets: []*sdknetwork.Subnet{
						{
							ID: &masterSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								AddressPrefix: pointerutils.ToPtr("10.0.0.0/24"),
								NetworkSecurityGroup: &sdknetwork.SecurityGroup{
									ID: &masterNSGv1,
								},
								ProvisioningState: pointerutils.ToPtr(sdknetwork.ProvisioningStateSucceeded),
							},
						},
					},
				},
			}

			if tt.modifyOC != nil {
				tt.modifyOC(oc)
			}
			vnetClient := mock_armnetwork.NewMockVirtualNetworksClient(controller)
			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, sdknetwork.VirtualNetworksClientGetResponse{VirtualNetwork: vnet})
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

var (
	validTestToken, _ = token.CreateTestToken(dummyObjectId, nil)
	mockAccessToken   = azcore.AccessToken{
		Token:     validTestToken, // mocked up data
		ExpiresOn: time.Now(),
	}
)

var (
	platformIdentity1SubnetActions = []string{
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/virtualNetworks/subnets/read",
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
	validCspVnetAuthorizationDecisions = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/read",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/write",
				AccessDecision: client.Allowed,
			},
		},
	}

	validFpspVnetAuthorizationDecisions = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/read",
				AccessDecision: client.Allowed,
			},
		},
	}

	invalidFpspVnetAuthorizationDecisionsReadNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/read",
				AccessDecision: client.NotAllowed,
			},
		},
	}

	validSubnetAuthorizationDecisions = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/read",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/write",
				AccessDecision: client.Allowed,
			},
		},
	}

	invalidCspVnetAuthorizationDecisionsReadNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/read",
				AccessDecision: client.NotAllowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/write",
				AccessDecision: client.Allowed,
			},
		},
	}

	invalidSubnetsAuthorizationDecisionsReadNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/read",
				AccessDecision: client.NotAllowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/write",
				AccessDecision: client.Allowed,
			},
		},
	}

	invalidCspVnetAuthorizationDecisionsMissingWrite = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/read",
				AccessDecision: client.Allowed,
			},
			// deliberately missing vnets write
		},
	}

	invalidSubnetsAuthorizationDecisionsMissingWrite = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/virtualNetworks/subnets/read",
				AccessDecision: client.Allowed,
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
		validatingFpsp      bool
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		mocks               func(*mock_env.MockInterface, *mock_azcore.MockTokenCredential, *mock_checkaccess.MockRemotePDPClient, context.CancelFunc)
		wantErr             string
	}{
		{
			name:           "pass: FPSP validation for CSP cluster",
			validatingFpsp: true,
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, _ context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validFpspVnetAuthorizationDecisions, nil)
			},
		},
		{
			name:           "fail: FPSP validation for CSP cluster - missing permissions",
			validatingFpsp: true,
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidFpspVnetAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on vnet '" + vnetID + "'.",
		},
		{
			name: "pass: CSP validation for CSP cluster",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, _ context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validCspVnetAuthorizationDecisions, nil)
			},
		},
		{
			name:               "pass - WI validation for MIWI Cluster",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, _ context.CancelFunc) {
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validCspVnetAuthorizationDecisions, nil)
			},
		},
		{
			name:               "Success - WI validation for MIWI Cluster - No intersecting VNET Actions",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActionsNoIntersect,
			},
		},
		{
			name: "fail: CSP validation for CSP cluster - missing permissions",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspVnetAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on vnet '" + vnetID + "'.",
		},
		{
			name:               "Fail - WI validation forMIWI Cluster - missing permissions",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspVnetAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on vnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet'.",
		},
		{
			name: "fail: CSP validation for CSP cluster - CheckAccess Return less entries than requested",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspVnetAuthorizationDecisionsMissingWrite, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on vnet '" + vnetID + "'.",
		},
		{
			name: "fail: CSP validation for CSP cluster - getting an invalid token from AAD",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, _ *mock_checkaccess.MockRemotePDPClient, _ context.CancelFunc) {
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				tokenCred.EXPECT().GetToken(gomock.Any(), gomock.Any()).
					Return(azcore.AccessToken{}, nil)
			},
			wantErr: "token contains an invalid number of segments",
		},
		{
			name: "fail: CSP validation for CSP cluster - getting an error when calling CheckAccessV2",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
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
			name: "fail: CSP validation for CSP cluster - getting a nil response from CheckAccessV2",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(nil, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on vnet '" + vnetID + "'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			tokenCred := mock_azcore.NewMockTokenCredential(controller)
			env := mock_env.NewMockInterface(controller)
			pdpClient := mock_checkaccess.NewMockRemotePDPClient(controller)
			if tt.mocks != nil {
				tt.mocks(env, tokenCred, pdpClient, cancel)
			}

			dv := &dynamic{
				env:                        env,
				appID:                      pointerutils.ToPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
				authorizerType:             AuthorizerClusterServicePrincipal,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
				pdpClient:                  pdpClient,
				checkAccessSubjectInfoCred: tokenCred,
			}

			if tt.validatingFpsp {
				dv.authorizerType = AuthorizerFirstParty
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

func TestValidateSubnetPermissions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                string
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		mocks               func(*mock_env.MockInterface, *mock_azcore.MockTokenCredential, *mock_checkaccess.MockRemotePDPClient, context.CancelFunc)
		wantErr             string
	}{
		{
			name: "pass: CSP validation for CSP cluster",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, _ context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validSubnetAuthorizationDecisions, nil)
			},
		},
		{
			name:               "pass - WI validation for MIWI Cluster",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, _ context.CancelFunc) {
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validSubnetAuthorizationDecisions, nil)
			},
		},
		{
			name:               "Success - WI validation for MIWI Cluster - No intersecting Subnet Actions",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActionsNoIntersect,
			},
		},
		{
			name: "fail: CSP validation for CSP cluster - missing permissions",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidSubnetsAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on subnet '" + masterSubnet + "'.",
		},
		{
			name:               "Fail - WI validation for MIWI Cluster - missing permissions",
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidSubnetsAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on subnet '" + masterSubnet + "'.",
		},
		{
			name: "fail: CSP validation for CSP cluster - CheckAccess Return less entries than requested",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidSubnetsAuthorizationDecisionsMissingWrite, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on subnet '" + masterSubnet + "'.",
		},
		{
			name: "fail: CSP validation for CSP cluster - getting an invalid token from AAD",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, _ *mock_checkaccess.MockRemotePDPClient, _ context.CancelFunc) {
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				tokenCred.EXPECT().GetToken(gomock.Any(), gomock.Any()).
					Return(azcore.AccessToken{}, nil)
			},
			wantErr: "token contains an invalid number of segments",
		},
		{
			name: "fail: CSP validation for CSP cluster - getting an error when calling CheckAccessV2",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
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
			name: "fail: CSP validation for CSP cluster - getting a nil response from CheckAccessV2",
			mocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(nil, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on subnet '" + masterSubnet + "'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			tokenCred := mock_azcore.NewMockTokenCredential(controller)
			env := mock_env.NewMockInterface(controller)
			pdpClient := mock_checkaccess.NewMockRemotePDPClient(controller)
			if tt.mocks != nil {
				tt.mocks(env, tokenCred, pdpClient, cancel)
			}

			dv := &dynamic{
				env:                        env,
				appID:                      pointerutils.ToPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
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

			subnetr := Subnet{ID: masterSubnet}

			err := dv.validateSubnetPermissions(ctx, subnetr)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

var (
	invalidCspRouteTablesAuthorizationDecisionsWriteNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/routeTables/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/read",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/write",
				AccessDecision: client.NotAllowed,
			},
		},
	}
	invalidCspRouteTablesAuthorizationDecisionsMissingWrite = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/routeTables/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/read",
				AccessDecision: client.Allowed,
			},
			// deliberately missing routetables write
		},
	}
	validCspRouteTablesAuthorizationDecisions = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/routeTables/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/read",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/write",
				AccessDecision: client.Allowed,
			},
		},
	}
	invalidFpspRouteTablesAuthorizationDecisionsJoinNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/routeTables/join/action",
				AccessDecision: client.NotAllowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/read",
				AccessDecision: client.Allowed,
			},
		},
	}
	validFpspRouteTablesAuthorizationDecisions = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/routeTables/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/routeTables/read",
				AccessDecision: client.Allowed,
			},
		},
	}
)

func TestValidateRouteTablesPermissions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                string
		validatingFpsp      bool
		subnet              Subnet
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		pdpClientMocks      func(*mock_env.MockInterface, *mock_azcore.MockTokenCredential, *mock_checkaccess.MockRemotePDPClient, context.CancelFunc)
		vnetMocks           func(*mock_armnetwork.MockVirtualNetworksClient, sdknetwork.VirtualNetworksClientGetResponse)
		wantErr             string
	}{
		{
			name:           "pass: FPSP validation for CSP cluster",
			validatingFpsp: true,
			subnet:         Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validFpspRouteTablesAuthorizationDecisions, nil)
			},
		},
		{
			name:           "fail: CSP validation for CSP cluster - permissions don't exist",
			validatingFpsp: true,
			subnet:         Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidFpspRouteTablesAuthorizationDecisionsJoinNotAllowed, nil)
			},
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on route table '" + workerRtID + "'.",
		},
		{
			name:   "fail: CSP validation for CSP cluster -failed to get vnet",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, errors.New("failed to get vnet"))
			},
			wantErr: "failed to get vnet",
		},
		{
			name:   "fail: CSP validation for CSP cluster - master subnet doesn't exist",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name:   "fail: CSP validation for CSP cluster - worker subnet ID doesn't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[1].ID = pointerutils.ToPtr("not valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + workerSubnet + "' could not be found.",
		},
		{
			name:   "pass: CSP validation for CSP cluster - no route table to check",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.RouteTable = nil
				vnet.Properties.Subnets[1].Properties.RouteTable = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
		{
			name:   "fail: CSP validation for CSP cluster - permissions don't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspRouteTablesAuthorizationDecisionsWriteNotAllowed, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on route table '" + workerRtID + "'.",
		},
		{
			name:               "Fail - WI validation for MIWI Cluster - permissions don't exist",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspRouteTablesAuthorizationDecisionsWriteNotAllowed, nil)
			},
			wantErr: "400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on route table '" + workerRtID + "'.",
		},
		{
			name:   "fail: CSP validation for CSP cluster - CheckAccessV2 doesn't return all the entries",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspRouteTablesAuthorizationDecisionsMissingWrite, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on route table '" + workerRtID + "'.",
		},
		{
			name:   "pass: CSP validation for CSP cluster",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validCspRouteTablesAuthorizationDecisions, nil)
			},
		},
		{
			name:               "pass - WI validation for MIWI Cluster",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validCspRouteTablesAuthorizationDecisions, nil)
			},
		},
		{
			name:               "Success - WI validation for MIWI Cluster - No intersecting Subnet Actions",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActionsNoIntersect,
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			tokenCred := mock_azcore.NewMockTokenCredential(controller)
			env := mock_env.NewMockInterface(controller)
			pdpClient := mock_checkaccess.NewMockRemotePDPClient(controller)
			vnetClient := mock_armnetwork.NewMockVirtualNetworksClient(controller)

			vnet := &sdknetwork.VirtualNetwork{
				ID: &vnetID,
				Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
					Subnets: []*sdknetwork.Subnet{
						{
							ID: &masterSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								RouteTable: &sdknetwork.RouteTable{
									ID: &masterRtID,
								},
							},
						},
						{
							ID: &workerSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								RouteTable: &sdknetwork.RouteTable{
									ID: &workerRtID,
								},
							},
						},
					},
				},
			}

			dv := &dynamic{
				appID:                      pointerutils.ToPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
				env:                        env,
				authorizerType:             AuthorizerClusterServicePrincipal,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
				checkAccessSubjectInfoCred: tokenCred,
				pdpClient:                  pdpClient,
				virtualNetworks:            vnetClient,
			}

			if tt.validatingFpsp {
				dv.authorizerType = AuthorizerFirstParty
			}

			if tt.pdpClientMocks != nil {
				tt.pdpClientMocks(env, tokenCred, pdpClient, cancel)
			}

			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, sdknetwork.VirtualNetworksClientGetResponse{VirtualNetwork: *vnet})
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
	invalidCspNatGWAuthorizationDecisionsReadNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/natGateways/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/read",
				AccessDecision: client.NotAllowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/write",
				AccessDecision: client.Allowed,
			},
		},
	}
	invalidCspNatGWAuthorizationDecisionsMissingWrite = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/natGateways/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/read",
				AccessDecision: client.Allowed,
			},
			// deliberately missing natGateway write
		},
	}
	validCspNatGWAuthorizationDecision = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/natGateways/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/read",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/natGateways/write",
				AccessDecision: client.Allowed,
			},
		},
	}
	invalidFpspNatGWAuthorizationDecisionsJoinNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/natGateways/join/action",
				AccessDecision: client.NotAllowed,
			},
		},
	}
	validFpspNatGWAuthorizationDecisions = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/natGateways/join/action",
				AccessDecision: client.Allowed,
			},
		},
	}
)

func TestValidateNatGatewaysPermissions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                string
		validatingFpsp      bool
		subnet              Subnet
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		pdpClientMocks      func(*mock_env.MockInterface, *mock_azcore.MockTokenCredential, *mock_checkaccess.MockRemotePDPClient, context.CancelFunc)
		vnetMocks           func(*mock_armnetwork.MockVirtualNetworksClient, sdknetwork.VirtualNetworksClientGetResponse)
		wantErr             string
	}{
		{
			name:           "pass: FPSP validation for CSP cluster",
			validatingFpsp: true,
			subnet:         Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validFpspNatGWAuthorizationDecisions, nil)
			},
		},
		{
			name:           "fail: FPSP validation for CSP cluster - permissions don't exist",
			validatingFpsp: true,
			subnet:         Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.
					EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidFpspNatGWAuthorizationDecisionsJoinNotAllowed, nil)
			},
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on nat gateway '" + workerNgID + "'.",
		},
		{
			name:   "fail: CSP validation for CSP cluster - failed to get vnet",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, errors.New("failed to get vnet"))
			},
			wantErr: "failed to get vnet",
		},
		{
			name:   "fail: CSP validation for CSP cluster - master subnet doesn't exist",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + masterSubnet + "' could not be found.",
		},
		{
			name:   "fail: CSP validation for CSP cluster - worker subnet ID doesn't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[1].ID = pointerutils.ToPtr("not valid")
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + workerSubnet + "' could not be found.",
		},
		{
			name:   "fail: CSP validation for CSP cluster - permissions don't exist",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.
					EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspNatGWAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on nat gateway '" + workerNgID + "'.",
		},
		{
			name:               "Fail - WI validation for MIWI Cluster - permissions don't exist",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				pdpClient.
					EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspNatGWAuthorizationDecisionsReadNotAllowed, nil)
			},
			wantErr: "400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on nat gateway '" + workerNgID + "'.",
		},
		{
			name:   "fail: CSP validation for CSP cluster - CheckAccessV2 doesn't return all permissions",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.
					EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						cancel()
					}).
					Return(&invalidCspNatGWAuthorizationDecisionsMissingWrite, nil)
			},
			wantErr: "400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on nat gateway '" + workerNgID + "'.",
		},
		{
			name:   "pass: CSP validation for CSP cluster",
			subnet: Subnet{ID: workerSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{}, nil)
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validCspNatGWAuthorizationDecision, nil)
			},
		},
		{
			name:               "pass - WI validation for MIWI Cluster",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
			pdpClientMocks: func(env *mock_env.MockInterface, tokenCred *mock_azcore.MockTokenCredential, pdpClient *mock_checkaccess.MockRemotePDPClient, cancel context.CancelFunc) {
				pdpClient.EXPECT().
					CheckAccess(gomock.Any(), gomock.Any()).
					Return(&validCspNatGWAuthorizationDecision, nil)
			},
		},
		{
			name:               "Success - WI validation for MIWI Cluster - No intersecting Subnet Actions",
			subnet:             Subnet{ID: workerSubnet},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActionsNoIntersect,
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
		{
			name:   "pass: no nat gateway to check",
			subnet: Subnet{ID: masterSubnet},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnet.Properties.Subnets[0].Properties.NatGateway = nil
				vnet.Properties.Subnets[1].Properties.NatGateway = nil
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					Return(vnet, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			vnetClient := mock_armnetwork.NewMockVirtualNetworksClient(controller)
			env := mock_env.NewMockInterface(controller)
			tokenCred := mock_azcore.NewMockTokenCredential(controller)
			pdpClient := mock_checkaccess.NewMockRemotePDPClient(controller)

			vnet := &sdknetwork.VirtualNetwork{
				ID: &vnetID,
				Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
					Subnets: []*sdknetwork.Subnet{
						{
							ID: &masterSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								NatGateway: &sdknetwork.SubResource{
									ID: &masterNgID,
								},
							},
						},
						{
							ID: &workerSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								NatGateway: &sdknetwork.SubResource{
									ID: &workerNgID,
								},
							},
						},
					},
				},
			}

			dv := &dynamic{
				appID:                      pointerutils.ToPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
				env:                        env,
				authorizerType:             AuthorizerClusterServicePrincipal,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
				checkAccessSubjectInfoCred: tokenCred,
				pdpClient:                  pdpClient,
				virtualNetworks:            vnetClient,
			}

			if tt.validatingFpsp {
				dv.authorizerType = AuthorizerFirstParty
			}

			if tt.pdpClientMocks != nil {
				tt.pdpClientMocks(env, tokenCred, pdpClient, cancel)
			}

			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, sdknetwork.VirtualNetworksClientGetResponse{VirtualNetwork: *vnet})
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
	subnetWithNSG := &sdknetwork.Subnet{
		Properties: &sdknetwork.SubnetPropertiesFormat{
			NetworkSecurityGroup: &sdknetwork.SecurityGroup{
				ID: &masterNSGv1,
			},
		},
	}
	subnetWithoutNSG := &sdknetwork.Subnet{
		Properties: &sdknetwork.SubnetPropertiesFormat{},
	}

	for _, tt := range []struct {
		name       string
		subnetByID map[string]*sdknetwork.Subnet
		wantErr    string
	}{
		{
			name: "pass: all subnets are attached",
			subnetByID: map[string]*sdknetwork.Subnet{
				"A": subnetWithNSG,
				"B": subnetWithNSG,
			},
		},
		{
			name: "fail: no subnets are attached",
			subnetByID: map[string]*sdknetwork.Subnet{
				"A": subnetWithoutNSG,
				"B": subnetWithoutNSG,
			},
			wantErr: "400: InvalidLinkedVNet: : When the enable-preconfigured-nsg option is specified, both the master and worker subnets should have network security groups (NSG) attached to them before starting the cluster installation.",
		},
		{
			name: "fail: parts of the subnets are attached",
			subnetByID: map[string]*sdknetwork.Subnet{
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
	validCspNsgAuthorizationDecisions = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/networkSecurityGroups/join/action",
				AccessDecision: client.Allowed,
			},
		},
	}

	invalidCspNsgAuthorizationDecisionsJoinNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/networkSecurityGroups/join/action",
				AccessDecision: client.NotAllowed,
			},
		},
	}

	validFpspNsgAuthorizationDecisions = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/networkSecurityGroups/join/action",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "Microsoft.Network/networkSecurityGroups/read",
				AccessDecision: client.Allowed,
			},
		},
	}

	invalidFpspNsgAuthorizationDecisionsJoinNotAllowed = client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "Microsoft.Network/networkSecurityGroups/join/action",
				AccessDecision: client.NotAllowed,
			},
			{
				ActionId:       "Microsoft.Network/networkSecurityGroups/read",
				AccessDecision: client.Allowed,
			},
		},
	}
)

func TestValidatePreconfiguredNSGPermissions(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name                string
		validatingFpsp      bool
		modifyOC            func(*api.OpenShiftCluster)
		platformIdentities  map[string]api.PlatformWorkloadIdentity
		platformIdentityMap map[string][]string
		checkAccessMocks    func(context.CancelFunc, *mock_checkaccess.MockRemotePDPClient, *mock_azcore.MockTokenCredential, *mock_env.MockInterface)
		vnetMocks           func(*mock_armnetwork.MockVirtualNetworksClient, sdknetwork.VirtualNetworksClientGetResponse)
		wantErrs            []string
	}{
		{
			name:           "pass: FPSP validation for CSP cluster",
			validatingFpsp: true,
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{
					Resource: client.ResourceInfo{Id: workerNSGv1},
				}, nil).AnyTimes()
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(_, _ interface{}) {
						cancel()
					}).
					Return(&validFpspNsgAuthorizationDecisions, nil).
					AnyTimes()
			},
		},
		{
			name:           "fail: FPSP validation for CSP cluster - missing permissions",
			validatingFpsp: true,
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{
					Resource: client.ResourceInfo{Id: workerNSGv1},
				}, nil).AnyTimes()
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(_, _ interface{}) {
						cancel()
					}).
					Return(&invalidFpspNsgAuthorizationDecisionsJoinNotAllowed, nil).
					AnyTimes()
			},
			wantErrs: []string{
				"400: InvalidResourceProviderPermissions: : The resource provider service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg'. This is required when the enable-preconfigured-nsg option is specified.",
				"400: InvalidResourceProviderPermissions: : The resource provider service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg'. This is required when the enable-preconfigured-nsg option is specified.",
			},
		},
		{
			name: "pass: skip when preconfiguredNSG is not enabled",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGDisabled
			},
		},
		{
			name: "fail: CSP validation for CSP cluster - doesn't have the permission on all NSGs",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{
					Resource: client.ResourceInfo{Id: masterNSGv1},
				}, nil).Times(1)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{
					Resource: client.ResourceInfo{Id: workerNSGv1},
				}, nil)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, authReq client.AuthorizationRequest) (*client.AuthorizationDecisionResponse, error) {
						cancel() // wait.PollImmediateUntil will always be invoked at least once
						switch authReq.Resource.Id {
						case masterNSGv1:
							return &validCspNsgAuthorizationDecisions, nil
						case workerNSGv1:
							return &invalidCspNsgAuthorizationDecisionsJoinNotAllowed, nil
						}
						return &invalidCspNsgAuthorizationDecisionsJoinNotAllowed, nil
					},
					).AnyTimes()
			},
			wantErrs: []string{
				"400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg'. This is required when the enable-preconfigured-nsg option is specified.",
				"400: InvalidServicePrincipalPermissions: : The cluster service principal (Application ID: fff51942-b1f9-4119-9453-aaa922259eb7) does not have required permissions on network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg'. This is required when the enable-preconfigured-nsg option is specified.",
			},
		},
		{
			name: "Fail - WI validation for MIWI Cluster - permissions don't exist on all nsg",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, authReq client.AuthorizationRequest) (*client.AuthorizationDecisionResponse, error) {
						cancel() // wait.PollImmediateUntil will always be invoked at least once
						switch authReq.Resource.Id {
						case masterNSGv1:
							return &validCspNsgAuthorizationDecisions, nil
						case workerNSGv1:
							return &invalidCspNsgAuthorizationDecisionsJoinNotAllowed, nil
						}
						return &invalidCspNsgAuthorizationDecisionsJoinNotAllowed, nil
					},
					).AnyTimes()
			},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
			wantErrs: []string{
				"400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg'. This is required when the enable-preconfigured-nsg option is specified.",
				"400: InvalidWorkloadIdentityPermissions: : The Dummy platform managed identity does not have required permissions on network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg'. This is required when the enable-preconfigured-nsg option is specified.",
			},
		},
		{
			name: "pass: CSP validation for CSP cluster",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					gomock.Any(), gomock.Any(), gomock.Any()).Return(&client.AuthorizationRequest{
					Resource: client.ResourceInfo{Id: workerNSGv1},
				}, nil).AnyTimes()
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(_, _ interface{}) {
						cancel()
					}).
					Return(&validCspNsgAuthorizationDecisions, nil).
					AnyTimes()
			},
		},
		{
			name: "pass - WI validation for MIWI Cluster",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(_, _ interface{}) {
						cancel()
					}).
					Return(&validCspNsgAuthorizationDecisions, nil).
					AnyTimes()
			},
			platformIdentities: platformIdentities,
			platformIdentityMap: map[string][]string{
				"Dummy": platformIdentity1SubnetActions,
			},
		},
		{
			name: "Success - WI validation for MIWI Cluster - No intersecting Subnet Actions",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGEnabled
			},
			vnetMocks: func(vnetClient *mock_armnetwork.MockVirtualNetworksClient, vnet sdknetwork.VirtualNetworksClientGetResponse) {
				vnetClient.EXPECT().
					Get(gomock.Any(), resourceGroupName, vnetName, nil).
					AnyTimes().
					Return(vnet, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				pdpClient.EXPECT().CheckAccess(gomock.Any(), gomock.Any()).
					Do(func(_, _ interface{}) {
						cancel()
					}).
					Return(&validCspNsgAuthorizationDecisions, nil).
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
			vnet := sdknetwork.VirtualNetwork{
				ID: &vnetID,
				Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
					Subnets: []*sdknetwork.Subnet{
						{
							ID: &masterSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								AddressPrefix: pointerutils.ToPtr("10.0.0.0/24"),
								NetworkSecurityGroup: &sdknetwork.SecurityGroup{
									ID: &masterNSGv1,
								},
								ProvisioningState: pointerutils.ToPtr(sdknetwork.ProvisioningStateSucceeded),
							},
						},
						{
							ID: &workerSubnet,
							Properties: &sdknetwork.SubnetPropertiesFormat{
								AddressPrefix: pointerutils.ToPtr("10.0.1.0/24"),
								NetworkSecurityGroup: &sdknetwork.SecurityGroup{
									ID: &workerNSGv1,
								},
								ProvisioningState: pointerutils.ToPtr(sdknetwork.ProvisioningStateSucceeded),
							},
						},
					},
				},
			}

			if tt.modifyOC != nil {
				tt.modifyOC(oc)
			}

			vnetClient := mock_armnetwork.NewMockVirtualNetworksClient(controller)

			if tt.vnetMocks != nil {
				tt.vnetMocks(vnetClient, sdknetwork.VirtualNetworksClientGetResponse{VirtualNetwork: vnet})
			}

			env := mock_env.NewMockInterface(controller)
			tokenCred := mock_azcore.NewMockTokenCredential(controller)
			pdpClient := mock_checkaccess.NewMockRemotePDPClient(controller)

			if tt.checkAccessMocks != nil {
				tt.checkAccessMocks(cancel, pdpClient, tokenCred, env)
			}

			dv := &dynamic{
				env:                        env,
				appID:                      pointerutils.ToPtr("fff51942-b1f9-4119-9453-aaa922259eb7"),
				authorizerType:             AuthorizerClusterServicePrincipal,
				virtualNetworks:            vnetClient,
				pdpClient:                  pdpClient,
				checkAccessSubjectInfoCred: tokenCred,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
			}

			if tt.validatingFpsp {
				dv.authorizerType = AuthorizerFirstParty
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
			utilerror.AssertOneOfErrorMessages(t, err, tt.wantErrs)
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
