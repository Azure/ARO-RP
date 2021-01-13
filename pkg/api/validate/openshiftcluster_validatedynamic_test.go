package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
)

func TestValidateProviders(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_features.MockProvidersClient)
		wantErr string
	}{
		{
			name: "pass",
			mocks: func(providersClient *mock_features.MockProvidersClient) {
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
			name: "fail: compute not registered",
			mocks: func(providersClient *mock_features.MockProvidersClient) {
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
			name: "fail: storage missing",
			mocks: func(providersClient *mock_features.MockProvidersClient) {
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
			mocks: func(providersClient *mock_features.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return(nil, errors.New("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			providerClient := mock_features.NewMockProvidersClient(controller)

			tt.mocks(providerClient)

			dv := &openShiftClusterDynamicValidator{
				log:       logrus.NewEntry(logrus.StandardLogger()),
				providers: providerClient,
			}

			err := dv.validateProviders(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
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
			dv := &openShiftClusterDynamicValidator{
				log: logrus.NewEntry(logrus.StandardLogger()),
				oc: &api.OpenShiftCluster{
					Location: "eastus",
				},
			}

			vnet := &mgmtnetwork.VirtualNetwork{
				Location: to.StringPtr(tt.location),
			}

			err := dv.validateVnetLocation(ctx, vnet)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateSubnet(t *testing.T) {
	ctx := context.Background()

	resourceGroupID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup"
	vnetID := resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/testVnet"
	genericSubnet := vnetID + "/subnet/genericSubnet"
	masterNSGv1 := resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg"

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
			name: "pass (master subnet)",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.MasterProfile = api.MasterProfile{
					SubnetID: genericSubnet,
				}
			},
		},
		{
			name: "pass (cluster in creating provisioning status)",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup = nil
			},
		},
		{
			name: "fail: subnet doe not exist on vnet",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				vnet.Subnets = nil
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + genericSubnet + "' could not be found.",
		},
		{
			name: "fail: private link service network policies enabled on master subnet",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.MasterProfile = api.MasterProfile{
					SubnetID: genericSubnet,
				}
			},
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].PrivateLinkServiceNetworkPolicies = to.StringPtr("Enabled")
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + genericSubnet + "' is invalid: must have privateLinkServiceNetworkPolicies disabled.",
		},
		{
			name: "fail: container registry endpoint doesn't exist",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].ServiceEndpoints = nil
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + genericSubnet + "' is invalid: must have Microsoft.ContainerRegistry serviceEndpoint.",
		},
		{
			name: "fail: network provisioning state not succeeded",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*(*vnet.Subnets)[0].ServiceEndpoints)[0].ProvisioningState = mgmtnetwork.Failed
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + genericSubnet + "' is invalid: must have Microsoft.ContainerRegistry serviceEndpoint.",
		},
		{
			name: "fail: provisioning state creating: subnet has NSG",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ProvisioningState = api.ProvisioningStateCreating
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + genericSubnet + "' is invalid: must not have a network security group attached.",
		},
		{
			name: "fail: invalid architecture version returns no NSG",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ArchitectureVersion = 9001
			},
			wantErr: "unknown architecture version 9001",
		},
		{
			name: "fail: nsg id doesn't match expected",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup.ID = to.StringPtr("not matching")
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + genericSubnet + "' is invalid: must have network security group '" + masterNSGv1 + "' attached.",
		},
		{
			name: "fail: invalid subnet CIDR",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].AddressPrefix = to.StringPtr("not-valid")
			},
			wantErr: "invalid CIDR address: not-valid",
		},
		{
			name: "fail: too small subnet CIDR",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].AddressPrefix = to.StringPtr("10.0.0.0/28")
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet '" + genericSubnet + "' is invalid: must be /27 or larger.",
		},
	} {
		oc := &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: resourceGroupID,
				},
			},
		}
		vnet := &mgmtnetwork.VirtualNetwork{
			ID: &vnetID,
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				Subnets: &[]mgmtnetwork.Subnet{
					{
						ID: &genericSubnet,
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("10.0.0.0/24"),
							NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
								ID: &masterNSGv1,
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

		// purposefully hardcoding path to "" so it is not needed in the wantErr message
		_, err := dv.validateSubnet(ctx, vnet, "", genericSubnet)
		if err != nil && err.Error() != tt.wantErr ||
			err == nil && tt.wantErr != "" {
			t.Error(err)
		}
	}
}

func TestValidateCIDRRanges(t *testing.T) {
	ctx := context.Background()

	resourceGroupID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup"
	vnetID := resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/testVnet"
	masterSubnet := vnetID + "/subnet/masterSubnet"
	workerSubnet := vnetID + "/subnet/workerSubnet"
	masterNSGv1 := resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg"
	workerNSGv1 := resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg"

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
			name: "fail: conflicting ranges",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "10.0.0.0/24"
			},
			wantErr: "400: InvalidLinkedVNet: : The provided CIDRs must not overlap: '10.0.0.0/24 overlaps with 10.0.0.0/24'.",
		},
	} {
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

		vnet := &mgmtnetwork.VirtualNetwork{
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
								ID: &workerNSGv1,
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

		err := dv.validateCIDRRanges(ctx, vnet)
		if err != nil && err.Error() != tt.wantErr ||
			err == nil && tt.wantErr != "" {
			t.Error(err)
		}
	}
}
