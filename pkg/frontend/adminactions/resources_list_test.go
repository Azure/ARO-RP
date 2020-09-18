package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
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

func TestResourcesList(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		mocks          func(*test, *mock_features.MockResourcesClient, *mock_compute.MockVirtualMachinesClient, *mock_network.MockVirtualNetworksClient, *mock_network.MockRouteTablesClient)
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, resourcesClient *mock_features.MockResourcesClient, vmClient *mock_compute.MockVirtualMachinesClient, vNetClient *mock_network.MockVirtualNetworksClient, routeTablesClient *mock_network.MockRouteTablesClient) {

				resourcesClient.EXPECT().List(gomock.Any(), "resourceGroup eq 'test-cluster'", "", nil).Return([]mgmtfeatures.GenericResourceExpanded{
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

				vNetClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mgmtnetwork.VirtualNetwork{
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

				routeTablesClient.EXPECT().Get(gomock.Any(), "mockrg", "routetable1", "").Return(mgmtnetwork.RouteTable{
					ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1"),
					Name: to.StringPtr("routetable1"),
				}, nil)

				resourcesClient.EXPECT().GetByID(gomock.Any(), "/subscriptions/id", azureclient.APIVersions["Microsoft.Storage"]).Return(mgmtfeatures.GenericResource{
					Name:     to.StringPtr("storage"),
					ID:       to.StringPtr("/subscriptions/id"),
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					Location: to.StringPtr("eastus"),
				}, nil)

				vmClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mgmtcompute.VirtualMachine{
					ID:   to.StringPtr("/subscriptions/id"),
					Type: to.StringPtr("Microsoft.Compute/virtualMachines"),
					VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
						ProvisioningState: to.StringPtr("Succeeded"),
					},
				}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`[{"properties":{"dhcpOptions":{"dnsServers":[]},"subnets":[{"properties":{"routeTable":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","tags":null}},"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"}]},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mockrg/providers/Microsoft.Network/routeTables/routetable1","name":"routetable1"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
		{
			name:       "vnet get error", //Get resources should continue on error from vNetClient.Get()
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, resourcesClient *mock_features.MockResourcesClient, vmClient *mock_compute.MockVirtualMachinesClient, vNetClient *mock_network.MockVirtualNetworksClient, routeTablesClient *mock_network.MockRouteTablesClient) {

				resourcesClient.EXPECT().List(gomock.Any(), "resourceGroup eq 'test-cluster'", "", nil).Return([]mgmtfeatures.GenericResourceExpanded{
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

				vNetClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mgmtnetwork.VirtualNetwork{}, fmt.Errorf("Any error during Get, expecting a permissions error"))

				resourcesClient.EXPECT().GetByID(gomock.Any(), "/subscriptions/id", azureclient.APIVersions["Microsoft.Storage"]).Return(mgmtfeatures.GenericResource{
					Name:     to.StringPtr("storage"),
					ID:       to.StringPtr("/subscriptions/id"),
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					Location: to.StringPtr("eastus"),
				}, nil)

				vmClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mgmtcompute.VirtualMachine{
					ID:   to.StringPtr("/subscriptions/id"),
					Type: to.StringPtr("Microsoft.Compute/virtualMachines"),
					VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
						ProvisioningState: to.StringPtr("Succeeded"),
					},
				}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`[{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return("eastus")

			resourcesClient := mock_features.NewMockResourcesClient(controller)
			vmClient := mock_compute.NewMockVirtualMachinesClient(controller)
			vNetClient := mock_network.NewMockVirtualNetworksClient(controller)
			routeTablesClient := mock_network.NewMockRouteTablesClient(controller)
			tt.mocks(tt, resourcesClient, vmClient, vNetClient, routeTablesClient)

			a := adminactions{
				log: logrus.NewEntry(logrus.StandardLogger()),
				env: env,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
						},
						MasterProfile: api.MasterProfile{
							SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID),
						},
					},
				},

				resourcesClient:   resourcesClient,
				vmClient:          vmClient,
				vNetClient:        vNetClient,
				routeTablesClient: routeTablesClient,
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
