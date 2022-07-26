package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
)

var (
	subscriptionId    = "0000000-0000-0000-0000-000000000000"
	vnetResourceGroup = "vnet-rg"
	vnetName          = "vnet"
	subnetNameWorker  = "worker"
	subnetNameMaster  = "master"
	subnetIdWorker    = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
	subnetIdMaster    = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
)

func TestEnableServiceEndpoints(t *testing.T) {
	type testData struct {
		name          string
		oc            *api.OpenShiftCluster
		setMocks      func(subnetManagerMock *mock_subnet.MockManager, endpointsAdderMock *mock_subnet.MockEndpointsAdder, testData testData)
		expectedError string
	}

	ctx := context.Background()
	testCases := []testData{
		{
			name: "should not call to subnetManager.CreateOrUpdate() because all endpoints are already in the subnet",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
				},
			},
			setMocks: func(subnetManagerMock *mock_subnet.MockManager, endpointsAdderMock *mock_subnet.MockEndpointsAdder, testData testData) {
				subnet := &mgmtnetwork.Subnet{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
							{
								Service:           to.StringPtr("Microsoft.Storage"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
						},
					},
				}

				subnetIds := []string{testData.oc.Properties.MasterProfile.SubnetID}
				endpoints := []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"}
				subnets := []*mgmtnetwork.Subnet{subnet}

				subnetManagerMock.
					EXPECT().
					GetAll(gomock.Any(), subnetIds).
					Return(subnets, nil)

				endpointsAdderMock.
					EXPECT().
					AddEndpointsToSubnets(endpoints, subnets).
					Return(nil)

				subnetManagerMock.
					EXPECT().
					CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
		},
		{
			name: "should not call to subnetManager.CreateOrUpdate() because all endpoints are already in both subnets",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
					WorkerProfiles: []api.WorkerProfile{
						{SubnetID: subnetIdWorker},
					},
				},
			},
			setMocks: func(subnetManagerMock *mock_subnet.MockManager, endpointsAdderMock *mock_subnet.MockEndpointsAdder, testData testData) {
				subnetIds := []string{testData.oc.Properties.MasterProfile.SubnetID, testData.oc.Properties.WorkerProfiles[0].SubnetID}

				subnet := &mgmtnetwork.Subnet{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
							{
								Service:           to.StringPtr("Microsoft.Storage"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
						},
					},
				}

				secondSubnet := &mgmtnetwork.Subnet{
					ID: to.StringPtr(subnetIdWorker),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
							{
								Service:           to.StringPtr("Microsoft.Storage"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
						},
					},
				}

				subnets := []*mgmtnetwork.Subnet{subnet, secondSubnet}

				endpoints := []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"}

				subnetManagerMock.
					EXPECT().
					GetAll(gomock.Any(), subnetIds).
					Return(subnets, nil)

				endpointsAdderMock.
					EXPECT().
					AddEndpointsToSubnets(endpoints, subnets).
					Return(nil)

				subnetManagerMock.
					EXPECT().
					CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
		},
		{
			name: "should return error and not call to subnetManager.CreateOrUpdate() because worker profile has no subnet ID",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: subnetIdMaster,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							Name:     "worker_profile_name",
							SubnetID: "",
						},
					},
				},
			},
			setMocks: func(subnetManagerMock *mock_subnet.MockManager, endpointsAdderMock *mock_subnet.MockEndpointsAdder, testData testData) {
				subnetManagerMock.
					EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedError: fmt.Sprintf("WorkerProfile '%v' has no SubnetID; check that the corresponding MachineSet is valid", "worker_profile_name"),
		},
		{
			name: "should call to subnetManager.CreateOrUpdate() with the updated subnets (as the endpoints were missing in both subnets)",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: subnetIdMaster,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							SubnetID: subnetIdWorker,
						},
					},
				},
			},
			setMocks: func(subnetManagerMock *mock_subnet.MockManager, endpointsAdderMock *mock_subnet.MockEndpointsAdder, testData testData) {
				initialSubnet1 := &mgmtnetwork.Subnet{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{},
					},
				}

				initialSubnet2 := &mgmtnetwork.Subnet{
					ID: to.StringPtr(subnetIdWorker),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{},
					},
				}

				initialSubnets := []*mgmtnetwork.Subnet{initialSubnet1, initialSubnet2}

				updatedSubnet1 := &mgmtnetwork.Subnet{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:   to.StringPtr("Microsoft.ContainerRegistry"),
								Locations: &[]string{"*"},
							},
							{
								Service:   to.StringPtr("Microsoft.Storage"),
								Locations: &[]string{"*"},
							},
						},
					},
				}

				updatedSubnet2 := &mgmtnetwork.Subnet{
					ID: to.StringPtr(subnetIdWorker),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:   to.StringPtr("Microsoft.ContainerRegistry"),
								Locations: &[]string{"*"},
							},
							{
								Service:   to.StringPtr("Microsoft.Storage"),
								Locations: &[]string{"*"},
							},
						},
					},
				}

				updatedSubnets := []*mgmtnetwork.Subnet{updatedSubnet1, updatedSubnet2}

				endpoints := []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"}

				subnetIds := []string{testData.oc.Properties.MasterProfile.SubnetID, testData.oc.Properties.WorkerProfiles[0].SubnetID}

				subnetManagerMock.
					EXPECT().
					GetAll(gomock.Any(), subnetIds).
					Return(initialSubnets, nil)

				endpointsAdderMock.
					EXPECT().
					AddEndpointsToSubnets(endpoints, initialSubnets).
					Return(updatedSubnets)

				subnetManagerMock.
					EXPECT().
					CreateOrUpdate(gomock.Any(), *updatedSubnets[0].ID, updatedSubnets[0]).
					Return(nil)

				subnetManagerMock.
					EXPECT().
					CreateOrUpdate(gomock.Any(), *updatedSubnets[1].ID, updatedSubnets[1]).
					Return(nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnetManagerMock := mock_subnet.NewMockManager(controller)
			endpointsAdderMock := mock_subnet.NewMockEndpointsAdder(controller)

			tc.setMocks(subnetManagerMock, endpointsAdderMock, tc)

			m := &manager{
				subnet:         subnetManagerMock,
				endpointsAdder: endpointsAdderMock,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: tc.oc,
				},
			}

			err := m.enableServiceEndpoints(ctx)

			if err != nil && err.Error() != tc.expectedError || err == nil && tc.expectedError != "" {
				t.Fatalf("expected error '%v', but got '%v'", tc.expectedError, err)
			}
		})
	}
}
