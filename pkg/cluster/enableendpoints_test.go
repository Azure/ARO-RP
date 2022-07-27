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

func TestEnableServiceEndpointsShouldNotCall_SubnetManager_CreateOrUpdate_AsAllEndpointsAreAlreadyInTheSubnet(t *testing.T) {
	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
		},
	}

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

	subnetIds := []string{oc.Properties.MasterProfile.SubnetID}
	endpoints := []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"}
	subnets := []*mgmtnetwork.Subnet{subnet}

	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx := context.Background()

	subnetManagerMock := mock_subnet.NewMockManager(controller)
	subnetManagerMock.
		EXPECT().
		GetAll(ctx, subnetIds).
		Return(subnets, nil)

	subnetManagerMock.
		EXPECT().
		CreateOrUpdate(ctx, gomock.Any(), gomock.Any()).
		Times(0)

	endpointsAdderMock := mock_subnet.NewMockEndpointsAdder(controller)
	endpointsAdderMock.
		EXPECT().
		AddEndpointsToSubnets(endpoints, subnets).
		Return(nil)

	m := &manager{
		subnet:         subnetManagerMock,
		endpointsAdder: endpointsAdderMock,
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

func TestEnableServiceEndpointsShouldNotCall_SubnetManager_CreateOrUpdate_AsAllEndpointsAreAlreadyInBothSubnets(t *testing.T) {
	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
			WorkerProfiles: []api.WorkerProfile{
				{SubnetID: subnetIdWorker},
			},
		},
	}

	subnetIds := []string{oc.Properties.MasterProfile.SubnetID, oc.Properties.WorkerProfiles[0].SubnetID}

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

	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx := context.Background()

	subnetManagerMock := mock_subnet.NewMockManager(controller)
	subnetManagerMock.
		EXPECT().
		GetAll(ctx, subnetIds).
		Return(subnets, nil)

	subnetManagerMock.
		EXPECT().
		CreateOrUpdate(ctx, gomock.Any(), gomock.Any()).
		Times(0)

	endpointsAdderMock := mock_subnet.NewMockEndpointsAdder(controller)
	endpointsAdderMock.
		EXPECT().
		AddEndpointsToSubnets(endpoints, subnets).
		Return(nil)

	m := &manager{
		subnet:         subnetManagerMock,
		endpointsAdder: endpointsAdderMock,
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

func TestEnableServiceEndpointsShouldReturnErrorAndNotCall_SubnetManager_CreateOrUpdate_AsWorkerProfileHasNoSubnetID(t *testing.T) {
	oc := &api.OpenShiftCluster{
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
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx := context.Background()

	subnetManagerMock := mock_subnet.NewMockManager(controller)

	subnetManagerMock.
		EXPECT().
		Get(ctx, gomock.Any()).
		Times(0)

	m := &manager{
		subnet: subnetManagerMock,
		doc: &api.OpenShiftClusterDocument{
			OpenShiftCluster: oc,
		},
	}

	err := m.enableServiceEndpoints(ctx)
	expectedError := fmt.Sprintf("WorkerProfile '%v' has no SubnetID; check that the corresponding MachineSet is valid", "worker_profile_name")

	if (err != nil && err.Error() != expectedError) || (err == nil && expectedError != "") {
		t.Fatalf("expected error '%v', but got '%v'", expectedError, err)
	}
}

func TestEnableServiceEndpointsShouldCall_SubnetManager_CreateOrUpdate_WithTheUpdatedSubnetsAsTheEndpointsWereMissingInBothSubnets(t *testing.T) {
	oc := &api.OpenShiftCluster{
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
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

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

	subnetIds := []string{oc.Properties.MasterProfile.SubnetID, oc.Properties.WorkerProfiles[0].SubnetID}

	ctx := context.Background()

	subnetManagerMock := mock_subnet.NewMockManager(controller)
	subnetManagerMock.
		EXPECT().
		GetAll(ctx, subnetIds).
		Return(initialSubnets, nil)

	subnetManagerMock.
		EXPECT().
		CreateOrUpdate(ctx, *updatedSubnets[0].ID, updatedSubnets[0]).
		Return(nil)

	subnetManagerMock.
		EXPECT().
		CreateOrUpdate(ctx, *updatedSubnets[1].ID, updatedSubnets[1]).
		Return(nil)

	endpointsAdderMock := mock_subnet.NewMockEndpointsAdder(controller)
	endpointsAdderMock.
		EXPECT().
		AddEndpointsToSubnets(endpoints, initialSubnets).
		Return(updatedSubnets)

	m := &manager{
		subnet:         subnetManagerMock,
		endpointsAdder: endpointsAdderMock,
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
