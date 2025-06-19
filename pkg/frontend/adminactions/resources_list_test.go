package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utiljson "github.com/Azure/ARO-RP/test/util/json"
)

func validListByResourceGroupMock(resources *mock_features.MockResourcesClient) {
	resources.EXPECT().ListByResourceGroup(gomock.Any(), "test-cluster", "", "", nil).Return([]mgmtfeatures.GenericResourceExpanded{
		{
			Name: to.Ptr("vm-1"),
			ID:   to.Ptr("/subscriptions/id"),
			Type: to.Ptr("Microsoft.Compute/virtualMachines"),
		},
		{
			Name: to.Ptr("storage"),
			ID:   to.Ptr("/subscriptions/id"),
			Type: to.Ptr("Microsoft.Storage/storageAccounts"),
		},
	}, nil)

	resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/id", azureclient.APIVersion("Microsoft.Storage")).Return(mgmtfeatures.GenericResource{
		Name:     to.Ptr("storage"),
		ID:       to.Ptr("/subscriptions/id"),
		Type:     to.Ptr("Microsoft.Storage/storageAccounts"),
		Location: to.Ptr("eastus"),
	}, nil)
}

func validVirtualMachinesMock(virtualMachines *mock_compute.MockVirtualMachinesClient) {
	virtualMachines.EXPECT().Get(gomock.Any(), "test-cluster", "vm-1", mgmtcompute.InstanceView).Return(mgmtcompute.VirtualMachine{
		ID:   to.Ptr("/subscriptions/id"),
		Type: to.Ptr("Microsoft.Compute/virtualMachines"),
		VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
			ProvisioningState: to.Ptr("Succeeded"),
		},
	}, nil).AnyTimes()
}

func validVirtualNetworksMock(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, mockSubID string) {
	virtualNetworks.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", nil).Return(sdknetwork.VirtualNetworksClientGetResponse{
		VirtualNetwork: sdknetwork.VirtualNetwork{
			ID:   to.Ptr("/subscriptions/id"),
			Type: to.Ptr("Microsoft.Network/virtualNetworks"),
			Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
				DhcpOptions: &sdknetwork.DhcpOptions{
					DNSServers: []*string{},
				},
				Subnets: []*sdknetwork.Subnet{
					{
						ID: to.Ptr(fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID)),
						Properties: &sdknetwork.SubnetPropertiesFormat{
							RouteTable: &sdknetwork.RouteTable{
								ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"),
							},
							NetworkSecurityGroup: &sdknetwork.SecurityGroup{
								ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/networkSecurityGroups/test-nsg"),
							},
						},
					},
				},
			},
		},
	}, nil)

	routeTables.EXPECT().Get(gomock.Any(), "mockrg", "routetable1", nil).Return(sdknetwork.RouteTablesClientGetResponse{
		RouteTable: sdknetwork.RouteTable{
			ID:   to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"),
			Name: to.Ptr("routetable1"),
		},
	}, nil)
}

func validDiskEncryptionSetsMock(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
	diskEncryptionSets.EXPECT().Get(gomock.Any(), "test-cluster", "test-cluster-des").Return(mgmtcompute.DiskEncryptionSet{
		ID:   to.Ptr("/subscriptions/id"),
		Type: to.Ptr("Microsoft.Compute/diskEncryptionSets"),
	}, nil)
}

func networkSecurityGroupMock(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, securityGroups *mock_armnetwork.MockSecurityGroupsClient) {
	virtualNetworks.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", nil).Return(sdknetwork.VirtualNetworksClientGetResponse{
		VirtualNetwork: sdknetwork.VirtualNetwork{
			ID:   to.Ptr("/subscriptions/id"),
			Type: to.Ptr("Microsoft.Network/virtualNetworks"),
			Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
				DhcpOptions: &sdknetwork.DhcpOptions{
					DNSServers: []*string{},
				},
				Subnets: []*sdknetwork.Subnet{
					{
						ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"),
						Properties: &sdknetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &sdknetwork.SecurityGroup{
								ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/networkSecurityGroups/test-nsg"),
							},
						},
					},
					{
						ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker1"),
						Properties: &sdknetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &sdknetwork.SecurityGroup{
								ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/byo-rg/providers/Microsoft.Network/networkSecurityGroups/byo-nsg"),
							},
						},
					},
					{
						ID:         to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker2"),
						Properties: &sdknetwork.SubnetPropertiesFormat{},
					},
				},
			},
		},
	}, nil)
	securityGroups.EXPECT().Get(gomock.Any(), "byo-rg", "byo-nsg", nil).Return(sdknetwork.SecurityGroupsClientGetResponse{
		SecurityGroup: sdknetwork.SecurityGroup{
			ID:   to.Ptr("/subscriptions/id"),
			Type: to.Ptr("Microsoft.Network/networkSecurityGroups"),
			Name: to.Ptr("byo-nsg"),
			Properties: &sdknetwork.SecurityGroupPropertiesFormat{
				Subnets: []*sdknetwork.Subnet{
					{
						ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/byo-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker1"),
					},
				},
			},
		},
	}, nil)
}

func TestResourcesList(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name                string
		mocks               func(*mock_armnetwork.MockVirtualNetworksClient, *mock_armnetwork.MockRouteTablesClient, *mock_compute.MockDiskEncryptionSetsClient, *mock_armnetwork.MockSecurityGroupsClient)
		wantResponse        []byte
		wantError           string
		diskEncryptionSetId string
	}

	for _, tt := range []*test{
		{
			name: "basic coverage",
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, securityGroups *mock_armnetwork.MockSecurityGroupsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)
			},
			wantResponse: []byte(`[{"apiVersion":"","properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"},"networkSecurityGroup":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/networkSecurityGroups/test-nsg"}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"apiVersion":"","id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "vnet get error", //Get resources should continue on error from virtualNetworks.Get()
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, securityGroups *mock_armnetwork.MockSecurityGroupsClient) {
				// Fail virtualNetworks with a GET error
				virtualNetworks.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", nil).Return(sdknetwork.VirtualNetworksClientGetResponse{}, fmt.Errorf("Any error during Get, expecting a permissions error"))
			},
			wantResponse: []byte(`[{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "enabled diskencryptionsets",
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, securityGroups *mock_armnetwork.MockSecurityGroupsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)
				validDiskEncryptionSetsMock(diskEncryptionSets)
			},
			diskEncryptionSetId: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-cluster-des", mockSubID),
			wantResponse:        []byte(`[{"apiVersion":"","properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"},"networkSecurityGroup":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/networkSecurityGroups/test-nsg"}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"apiVersion":"","id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"id":"/subscriptions/id","type":"Microsoft.Compute/diskEncryptionSets"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "error getting diskencryptionsets",
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, securityGroups *mock_armnetwork.MockSecurityGroupsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)

				// Fail diskEncryptionSets with a GET error
				diskEncryptionSets.EXPECT().Get(gomock.Any(), "test-cluster", "test-cluster-des").Return(mgmtcompute.DiskEncryptionSet{}, fmt.Errorf("Any error during Get, expecting a permissions error"))
			},
			diskEncryptionSetId: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-cluster-des", mockSubID),
			wantResponse:        []byte(`[{"apiVersion":"","properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"},"networkSecurityGroup":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/networkSecurityGroups/test-nsg"}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"apiVersion":"","id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "get BYO NetworkSecurityGroup",
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient, securityGroups *mock_armnetwork.MockSecurityGroupsClient) {
				networkSecurityGroupMock(virtualNetworks, securityGroups)
			},
			wantResponse: []byte(`[{"apiVersion":"","id":"/subscriptions/id","properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master","properties":{"networkSecurityGroup":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/networkSecurityGroups/test-nsg"}}},{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker1","properties":{"networkSecurityGroup":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/byo-rg/providers/Microsoft.Network/networkSecurityGroups/byo-nsg"}}},{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker2","properties":{}}]},"type":"Microsoft.Network/virtualNetworks"},{"apiVersion":"","id":"/subscriptions/id","name":"byo-nsg","properties":{"subnets":[{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/byo-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker1"}]},"type":"Microsoft.Network/networkSecurityGroups"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return("eastus")

			resources := mock_features.NewMockResourcesClient(controller)
			virtualMachines := mock_compute.NewMockVirtualMachinesClient(controller)
			virtualNetworks := mock_armnetwork.NewMockVirtualNetworksClient(controller)
			routeTables := mock_armnetwork.NewMockRouteTablesClient(controller)
			diskEncryptionSets := mock_compute.NewMockDiskEncryptionSetsClient(controller)
			networkSecurityGroups := mock_armnetwork.NewMockSecurityGroupsClient(controller)

			validListByResourceGroupMock(resources)
			validVirtualMachinesMock(virtualMachines)

			tt.mocks(virtualNetworks, routeTables, diskEncryptionSets, networkSecurityGroups)

			a := azureActions{
				log: logrus.NewEntry(logrus.StandardLogger()),
				env: env,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
						},
						MasterProfile: api.MasterProfile{
							SubnetID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID),
							DiskEncryptionSetID: tt.diskEncryptionSetId,
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker1", mockSubID),
							},
							{
								SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker2", mockSubID),
							},
						},
					},
				},

				resources:          resources,
				virtualMachines:    virtualMachines,
				virtualNetworks:    virtualNetworks,
				diskEncryptionSets: diskEncryptionSets,
				routeTables:        routeTables,
				securityGroups:     networkSecurityGroups,
			}

			reader, writer := io.Pipe()
			err := a.WriteToStream(ctx, writer)

			b, _ := io.ReadAll(reader)
			if tt.wantError == "" {
				if tt.wantResponse != nil {
					utiljson.AssertJsonMatches(t, tt.wantResponse, b)
				}
			} else {
				if err.Error() != tt.wantError {
					t.Fatal(err)
				}
			}
		})
	}
}
