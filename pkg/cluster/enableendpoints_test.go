package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

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

func TestEnableServiceEndpointsShouldCall_SubnetManager_CreateOrUpdateFromIds_AndReturnNoError(t *testing.T) {
	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
			WorkerProfiles: []api.WorkerProfile{
				{SubnetID: subnetIdWorker},
			},
		},
	}

	subnetIds := []string{oc.Properties.MasterProfile.SubnetID, oc.Properties.WorkerProfiles[0].SubnetID}

	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx := context.Background()

	subnetManagerMock := mock_subnet.NewMockManager(controller)

	subnetManagerMock.
		EXPECT().
		CreateOrUpdateFromIds(ctx, subnetIds).
		Return(nil)

	m := &manager{
		subnet: subnetManagerMock,
		doc: &api.OpenShiftClusterDocument{
			OpenShiftCluster: oc,
		},
	}

	err := m.enableServiceEndpoints(ctx)
	expectedError := ""

	if (err != nil && err.Error() != expectedError) || (err == nil && expectedError != "") {
		t.Fatalf("expected error '%v', but got '%v'", expectedError, err)
	}
}

func TestEnableServiceEndpointsShouldReturnWorkerProfileHasNoSubnetIdErrorAndShouldNotCall_SubnetManager_CreateOrUpdateFromIds(t *testing.T) {
	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			MasterProfile: api.MasterProfile{
				SubnetID: subnetIdMaster,
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					Name:     "profile_name",
					SubnetID: "",
				},
			},
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx := context.Background()

	subnetManagerMock := mock_subnet.NewMockManager(controller)

	subnetManagerMock.
		EXPECT().
		CreateOrUpdateFromIds(gomock.Any(), gomock.Any()).
		Times(0).
		Return(nil)

	m := &manager{
		subnet: subnetManagerMock,
		doc: &api.OpenShiftClusterDocument{
			OpenShiftCluster: oc,
		},
	}

	err := m.enableServiceEndpoints(ctx)
	expectedError := "WorkerProfile 'profile_name' has no SubnetID; check that the corresponding MachineSet is valid"

	if (err != nil && err.Error() != expectedError) || (err == nil && expectedError != "") {
		t.Fatalf("expected error '%v', but got '%v'", expectedError, err)
	}
}
