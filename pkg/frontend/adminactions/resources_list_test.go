package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"testing"

	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

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
	}, nil).AnyTimes()
}

func validVirtualNetworksMock(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, mockSubID string) {
	virtualNetworks.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", nil).Return(sdknetwork.VirtualNetworksClientGetResponse{
		VirtualNetwork: sdknetwork.VirtualNetwork{
			ID:   ptr.To("/subscriptions/id"),
			Type: ptr.To("Microsoft.Network/virtualNetworks"),
			Properties: &sdknetwork.VirtualNetworkPropertiesFormat{
				DhcpOptions: &sdknetwork.DhcpOptions{
					DNSServers: []*string{},
				},
				Subnets: []*sdknetwork.Subnet{
					{
						ID: ptr.To(fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID)),
						Properties: &sdknetwork.SubnetPropertiesFormat{
							RouteTable: &sdknetwork.RouteTable{
								ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"),
							},
						},
					},
				},
			},
		},
	}, nil)

	routeTables.EXPECT().Get(gomock.Any(), "mockrg", "routetable1", nil).Return(sdknetwork.RouteTablesClientGetResponse{
		RouteTable: sdknetwork.RouteTable{
			ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"),
			Name: ptr.To("routetable1"),
		},
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
		mocks               func(*mock_armnetwork.MockVirtualNetworksClient, *mock_armnetwork.MockRouteTablesClient, *mock_compute.MockDiskEncryptionSetsClient)
		wantResponse        []byte
		wantError           string
		diskEncryptionSetId string
	}

	for _, tt := range []*test{
		{
			name: "basic coverage",
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)
			},
			wantResponse: []byte(`[{"apiVersion":"","properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"apiVersion":"","id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "vnet get error", //Get resources should continue on error from virtualNetworks.Get()
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
				// Fail virtualNetworks with a GET error
				virtualNetworks.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", nil).Return(sdknetwork.VirtualNetworksClientGetResponse{}, fmt.Errorf("Any error during Get, expecting a permissions error"))
			},
			wantResponse: []byte(`[{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "enabled diskencryptionsets",
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)
				validDiskEncryptionSetsMock(diskEncryptionSets)
			},
			diskEncryptionSetId: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-cluster-des", mockSubID),
			wantResponse:        []byte(`[{"apiVersion":"","properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"apiVersion":"","id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"id":"/subscriptions/id","type":"Microsoft.Compute/diskEncryptionSets"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name: "error getting diskencryptionsets",
			mocks: func(virtualNetworks *mock_armnetwork.MockVirtualNetworksClient, routeTables *mock_armnetwork.MockRouteTablesClient, diskEncryptionSets *mock_compute.MockDiskEncryptionSetsClient) {
				validVirtualNetworksMock(virtualNetworks, routeTables, mockSubID)

				// Fail diskEncryptionSets with a GET error
				diskEncryptionSets.EXPECT().Get(gomock.Any(), "test-cluster", "test-cluster-des").Return(mgmtcompute.DiskEncryptionSet{}, fmt.Errorf("Any error during Get, expecting a permissions error"))
			},
			diskEncryptionSetId: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-cluster-des", mockSubID),
			wantResponse:        []byte(`[{"apiVersion":"","properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"apiVersion":"","id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
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
