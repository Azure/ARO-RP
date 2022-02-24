package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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
)

func getValidSubnet(endpoints bool, state *mgmtnetwork.ProvisioningState) *mgmtnetwork.Subnet {
	s := &mgmtnetwork.Subnet{
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{},
	}
	if endpoints {
		s.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{
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
		}
		if state != nil {
			for i := range *s.SubnetPropertiesFormat.ServiceEndpoints {
				(*s.SubnetPropertiesFormat.ServiceEndpoints)[i].ProvisioningState = *state
			}
		}
	}
	return s
}

func TestEnableServiceEndpoints(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name string
		oc   *api.OpenShiftCluster
		mock func(subnetMock *mock_subnet.MockManager, tt test)
	}

	for _, tt := range []test{
		{
			name: "nothing to do",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							SubnetID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker,
						},
					},
				},
			},
			mock: func(subnetClient *mock_subnet.MockManager, tt test) {
				subnets := []string{
					tt.oc.Properties.MasterProfile.SubnetID,
				}
				for _, wp := range tt.oc.Properties.WorkerProfiles {
					subnets = append(subnets, wp.SubnetID)
				}

				for _, subnetId := range subnets {
					state := mgmtnetwork.Succeeded
					subnetClient.EXPECT().Get(gomock.Any(), subnetId).Return(getValidSubnet(true, &state), nil)
				}
			},
		},
		{
			name: "enable endpoints",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							SubnetID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker,
						},
					},
				},
			},
			mock: func(subnetClient *mock_subnet.MockManager, tt test) {
				subnets := []string{
					tt.oc.Properties.MasterProfile.SubnetID,
				}
				for _, wp := range tt.oc.Properties.WorkerProfiles {
					subnets = append(subnets, wp.SubnetID)
				}

				for _, subnetId := range subnets {
					subnetClient.EXPECT().Get(gomock.Any(), subnetId).Return(getValidSubnet(false, nil), nil)
					subnetClient.EXPECT().CreateOrUpdate(gomock.Any(), subnetId, getValidSubnet(true, nil)).Return(nil)
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnetClient := mock_subnet.NewMockManager(controller)

			tt.mock(subnetClient, tt)

			m := &manager{
				subnet: subnetClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: tt.oc,
				},
			}

			// we don't test errors as all of them would be out of our control
			_ = m.enableServiceEndpoints(ctx)
		})
	}
}
