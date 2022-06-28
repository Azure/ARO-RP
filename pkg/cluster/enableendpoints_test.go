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
	ctx := context.Background()

	type test struct {
		name          string
		oc            *api.OpenShiftCluster
		setMocks      func(subnetManagerMock *mock_subnet.MockManager, subnetsUpdaterMock *mock_subnet.MockUpdater, tt test)
		expectedError string
	}

	tt := []test{
		{
			name: "should not call to subnetManager.CreateOrUpdate() because all endpoints are already in the subnet",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
				},
			},
			setMocks: func(subnetManagerMock *mock_subnet.MockManager, subnetsUpdaterMock *mock_subnet.MockUpdater, tt test) {
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

				endpoints := []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"}
				subnets := []*mgmtnetwork.Subnet{subnet}

				subnetManagerMock.
					EXPECT().
					Get(gomock.Any(), tt.oc.Properties.MasterProfile.SubnetID).
					Return(subnet, nil)

				subnetsUpdaterMock.
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
			setMocks: func(subnetManagerMock *mock_subnet.MockManager, subnetsUpdaterMock *mock_subnet.MockUpdater, tt test) {
				subnetIds := []string{tt.oc.Properties.MasterProfile.SubnetID, tt.oc.Properties.WorkerProfiles[0].SubnetID}

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
					Get(gomock.Any(), subnetIds[0]).
					Return(subnet, nil)

				subnetManagerMock.
					EXPECT().
					Get(gomock.Any(), subnetIds[1]).
					Return(secondSubnet, nil)

				subnetsUpdaterMock.
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
			setMocks: func(subnetManagerMock *mock_subnet.MockManager, subnetsUpdaterMock *mock_subnet.MockUpdater, tt test) {
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
			setMocks: func(subnetManagerMock *mock_subnet.MockManager, subnetsUpdaterMock *mock_subnet.MockUpdater, tt test) {
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

				subnetManagerMock.
					EXPECT().
					Get(gomock.Any(), subnetIdMaster).
					Return(initialSubnet1, nil)

				subnetManagerMock.
					EXPECT().
					Get(gomock.Any(), subnetIdWorker).
					Return(initialSubnet2, nil)

				subnetsUpdaterMock.
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

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnetManagerMock := mock_subnet.NewMockManager(controller)
			subnetsUpdaterMock := mock_subnet.NewMockUpdater(controller)

			tc.setMocks(subnetManagerMock, subnetsUpdaterMock, tc)

			m := &manager{
				subnet:         subnetManagerMock,
				subnetsUpdater: subnetsUpdaterMock,
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
