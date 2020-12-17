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
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mockrefreshable "github.com/Azure/ARO-RP/pkg/util/mocks/refreshable"
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
			name: "all registered",
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
			name: "compute not registered",
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
			name: "storage missing",
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

			err := validateProviders(ctx, logrus.NewEntry(logrus.StandardLogger()), providerClient)
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
	workerSubnet2 := vnetID + "/subnet/workerSubnet2"
	masterNSGv1 := resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg"
	workerNSGv1 := resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg"
	commonNSGv2 := resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-nsg"

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
				vnet.Location = to.StringPtr("westeurope")
			},
			wantErr: "400: InvalidLinkedVNet: : The vnet location 'westeurope' must match the cluster location 'eastus'.",
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
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles[0].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' could not be found.`,
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
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles[0].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must have Microsoft.ContainerRegistry serviceEndpoint.`,
		},
		{
			name: "invalid master nsg arch v1",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup = nil
			},
			wantErr: `400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/masterSubnet' is invalid: must have network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg' attached.`,
		},
		{
			name: "invalid master nsg arch v2",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup = nil
				(*vnet.Subnets)[1].NetworkSecurityGroup.ID = &commonNSGv2
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ArchitectureVersion = api.ArchitectureVersionV2
			},
			wantErr: `400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/masterSubnet' is invalid: must have network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-nsg' attached.`,
		},
		{
			name: "invalid worker nsg arch v1",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[1].NetworkSecurityGroup = nil
			},
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles[0].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must have network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg' attached.`,
		},
		{
			name: "invalid worker nsg arch v2",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[0].NetworkSecurityGroup.ID = &commonNSGv2
				(*vnet.Subnets)[1].NetworkSecurityGroup = nil
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.ArchitectureVersion = api.ArchitectureVersionV2
			},
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles[0].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must have network security group '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/networkSecurityGroups/aro-nsg' attached.`,
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
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles[0].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must not have a network security group attached.`,
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
			wantErr: `400: InvalidLinkedVNet: properties.workerProfiles[0].subnetId: The provided subnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet/subnet/workerSubnet' is invalid: must be /27 or larger.`,
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
			name: "two worker pools on the same subnet",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.WorkerProfiles = append(oc.Properties.WorkerProfiles, api.WorkerProfile{
					SubnetID: workerSubnet,
				})
			},
			wantErr: "",
		},
		{
			name: "two worker pools on the on two different subnets which do not overlap with pod or service cidr",
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.WorkerProfiles = append(oc.Properties.WorkerProfiles, api.WorkerProfile{
					SubnetID: workerSubnet2,
				})
			},
			wantErr: "",
		},
		{
			name: "two worker pools on the on two different subnets one of which overlaps with pod or service cidr",
			modifyVnet: func(vnet *mgmtnetwork.VirtualNetwork) {
				(*vnet.Subnets)[2].AddressPrefix = to.StringPtr("10.0.3.0/24")
			},
			modifyOC: func(oc *api.OpenShiftCluster) {
				oc.Properties.WorkerProfiles = append(oc.Properties.WorkerProfiles, api.WorkerProfile{
					SubnetID: workerSubnet2,
				})
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
				Location: "eastus",
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
				Location: to.StringPtr("eastus"),
				ID:       &vnetID,
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
						{
							ID: &workerSubnet2,
							SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
								AddressPrefix: to.StringPtr("10.0.4.0/24"),
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

			err := validateVnet(ctx, logrus.NewEntry(logrus.StandardLogger()), oc, vnet)
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

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_authorization.MockPermissionsClient, func())
		wantErr string
	}{
		{
			name: "pass",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
				permissionsClient.EXPECT().
					ListForResource(gomock.Any(), "", "", "", "", "").
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
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider does not have Network Contributor permission on vnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet'.",
		},
		{
			name: "fail: not found",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
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

			permissionsClient := mock_authorization.NewMockPermissionsClient(controller)

			tt.mocks(permissionsClient, cancel)

			err := validateVnetPermissions(ctx, logrus.NewEntry(logrus.StandardLogger()), mockrefreshable.NewMockAuthorizer(controller), permissionsClient, vnetID, &azure.Resource{}, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
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

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_authorization.MockPermissionsClient, func())
		vnet    func(*mgmtnetwork.VirtualNetwork)
		subnet  string
		wantErr string
	}{
		{
			name: "pass",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
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
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
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
			wantErr: "400: InvalidResourceProviderPermissions: : The resource provider does not have Network Contributor permission on route table '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/routeTables/testRT'.",
		},
		{
			name: "fail: not found",
			mocks: func(permissionsClient *mock_authorization.MockPermissionsClient, cancel func()) {
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

			permissionsClient := mock_authorization.NewMockPermissionsClient(controller)

			if tt.mocks != nil {
				tt.mocks(permissionsClient, cancel)
			}

			if tt.vnet != nil {
				tt.vnet(vnet)
			}

			err := validateRouteTablePermissionsSubnet(ctx, logrus.NewEntry(logrus.StandardLogger()), mockrefreshable.NewMockAuthorizer(controller), permissionsClient, vnet, tt.subnet, "properties.masterProfile.subnetId", api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
