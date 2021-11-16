package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func validListByResourceGroupMock(resources *mock_features.MockResourcesClient) {
	resources.EXPECT().ListByResourceGroup(gomock.Any(), "test-cluster", "", "", nil).Return([]mgmtfeatures.GenericResourceExpanded{
		{
			Name: to.StringPtr("vm-1"),
			ID:   to.StringPtr("/subscriptions/id"),
			Type: to.StringPtr("Microsoft.Compute/virtualMachines"),
		},
		{
			Name: to.StringPtr("storage"),
			ID:   to.StringPtr("/subscriptions/id"),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		},
	}, nil)

	resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/id", azureclient.APIVersion("Microsoft.Storage")).Return(mgmtfeatures.GenericResource{
		Name:     to.StringPtr("storage"),
		ID:       to.StringPtr("/subscriptions/id"),
		Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
		Location: to.StringPtr("eastus"),
	}, nil)
}

func validVirtualMachinesMock(virtualMachines *mock_compute.MockVirtualMachinesClient) {
	virtualMachines.EXPECT().Get(gomock.Any(), "test-cluster", "vm-1", mgmtcompute.InstanceView).Return(mgmtcompute.VirtualMachine{
		ID:   to.StringPtr("/subscriptions/id"),
		Type: to.StringPtr("Microsoft.Compute/virtualMachines"),
		VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
			ProvisioningState: to.StringPtr("Succeeded"),
		},
	}, nil)
}

func validVirtualNetworksMock(virtualNetworks *mock_network.MockVirtualNetworksClient, routeTables *mock_network.MockRouteTablesClient, mockSubID string) {
	virtualNetworks.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", "").Return(mgmtnetwork.VirtualNetwork{
		ID:   to.StringPtr("/subscriptions/id"),
		Type: to.StringPtr("Microsoft.Network/virtualNetworks"),
		VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
			DhcpOptions: &mgmtnetwork.DhcpOptions{
				DNSServers: &[]string{},
			},
			Subnets: &[]mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID)),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						RouteTable: &mgmtnetwork.RouteTable{
							ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"),
						},
					},
				},
			},
		},
	}, nil)

	routeTables.EXPECT().Get(gomock.Any(), "mockrg", "routetable1", "").Return(mgmtnetwork.RouteTable{
		ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"),
		Name: to.StringPtr("routetable1"),
	}, nil)
}

func validDiskEncryptionSetsMock(diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
	diskEncryptionSets.EXPECT().Get(gomock.Any(), "test-cluster", "test-cluster-des").Return(mgmtcompute.DiskEncryptionSet{
		ID:   to.StringPtr("/subscriptions/id"),
		Type: to.StringPtr("Microsoft.Compute/diskEncryptionSets"),
	}, nil)
}

func TestResourcesList(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name                string
		mocks               func(*mock_network.MockVirtualNetworksClient, *mock_network.MockRouteTablesClient, *mock_compute.MockDiskEncryptionSetsClient)
		wantResponse        []byte
		wantError           string
		diskEncryptionSetId string
	}

	for _, tt := range []*test{
		{
			name: "basic coverage",
			mocks: func(virtualNetworks *mock_network.MockVirtualNetworksClient, routeTables *mock_network.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)
			},
			wantResponse: []byte(`[{"properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","tags":null}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "vnet get error", //Get resources should continue on error from virtualNetworks.Get()
			mocks: func(virtualNetworks *mock_network.MockVirtualNetworksClient, routeTables *mock_network.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
				// Fail virtualNetworks with a GET error
				virtualNetworks.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", "").Return(mgmtnetwork.VirtualNetwork{}, fmt.Errorf("Any error during Get, expecting a permissions error"))
			},
			wantResponse: []byte(`[{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "enabled diskencryptionsets",
			mocks: func(virtualNetworks *mock_network.MockVirtualNetworksClient, routeTables *mock_network.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)
				validDiskEncryptionSetsMock(diskEncryptionSets)
			},
			diskEncryptionSetId: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-cluster-des", mockSubID),
			wantResponse:        []byte(`[{"properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","tags":null}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"id":"/subscriptions/id","type":"Microsoft.Compute/diskEncryptionSets"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "error getting diskencryptionsets",
			mocks: func(virtualNetworks *mock_network.MockVirtualNetworksClient, routeTables *mock_network.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)

				// Fail diskEncryptionSets with a GET error
				diskEncryptionSets.EXPECT().Get(gomock.Any(), "test-cluster", "test-cluster-des").Return(mgmtcompute.DiskEncryptionSet{}, fmt.Errorf("Any error during Get, expecting a permissions error"))
			},
			diskEncryptionSetId: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-cluster-des", mockSubID),
			wantResponse:        []byte(`[{"properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","tags":null}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return("eastus")

			resources := mock_features.NewMockResourcesClient(controller)
			virtualMachines := mock_compute.NewMockVirtualMachinesClient(controller)
			virtualNetworks := mock_network.NewMockVirtualNetworksClient(controller)
			routeTables := mock_network.NewMockRouteTablesClient(controller)
			diskEncryptionSets := mock_compute.NewMockDiskEncryptionSetsClient(controller)

			validListByResourceGroupMock(resources)
			validVirtualMachinesMock(virtualMachines)

			tt.mocks(virtualNetworks, routeTables, diskEncryptionSets)

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
					},
				},

				resources:          resources,
				virtualMachines:    virtualMachines,
				virtualNetworks:    virtualNetworks,
				diskEncryptionSets: diskEncryptionSets,
				routeTables:        routeTables,
			}

			b, err := a.ResourcesList(ctx)

			if tt.wantError == "" {
				if tt.wantResponse != nil {
					if !bytes.Equal(b, tt.wantResponse) {
						t.Error(string(b))
					}
				}
			} else {
				if err.Error() != tt.wantError {
					t.Fatal(err)
				}
			}
		})
	}
}
