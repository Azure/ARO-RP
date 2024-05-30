package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
)

var (
	subscriptionId    = "0000000-0000-0000-0000-000000000000"
	vnetResourceGroup = "vnet-rg"
	vnetName          = "vnet"
	subnetNameWorker  = "worker"
	subnetNameMaster  = "master"
	subnetIdWorker    = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnet/" + subnetNameWorker
	subnetIdMaster    = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnet/" + subnetNameMaster
	masterSubnet      = armnetwork.Subnet{
		ID: ptr.To(subnetIdMaster),
		Properties: &armnetwork.SubnetPropertiesFormat{
			ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
			ServiceEndpoints:  []*armnetwork.ServiceEndpointPropertiesFormat{},
		},
	}
	workerSubnet = armnetwork.Subnet{
		ID: ptr.To(subnetIdWorker),
		Properties: &armnetwork.SubnetPropertiesFormat{
			ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
			ServiceEndpoints:  []*armnetwork.ServiceEndpointPropertiesFormat{},
		},
	}
)

func TestEnsureServiceEndpoints(t *testing.T) {
	for _, tt := range []struct {
		name        string
		oc          *api.OpenShiftCluster
		mock        func(subnets *mock_armnetwork.MockSubnetsClient)
		expectedErr string
	}{
		{
			name: "It should do nothing when egress lockdown is enabled",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
					WorkerProfiles: []api.WorkerProfile{
						{SubnetID: subnetIdWorker},
					},
					FeatureProfile: api.FeatureProfile{
						GatewayEnabled: true,
					},
				},
			},
			mock: func(subnets *mock_armnetwork.MockSubnetsClient) {
				subnets.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name: "It should update subnet when egress lockdown is disabled",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
					WorkerProfiles: []api.WorkerProfile{
						{SubnetID: subnetIdWorker},
					},
				},
			},
			mock: func(subnets *mock_armnetwork.MockSubnetsClient) {
				subnets.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: masterSubnet}, nil)
				subnets.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: workerSubnet}, nil)
				expectedMasterSubnet := armnetwork.Subnet{
					ID: ptr.To(subnetIdMaster),
					Properties: &armnetwork.SubnetPropertiesFormat{
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:   ptr.To("Microsoft.ContainerRegistry"),
								Locations: []*string{ptr.To("*")},
							},
							{
								Service:   ptr.To("Microsoft.Storage"),
								Locations: []*string{ptr.To("*")},
							},
						},
					},
				}
				expectedWorkerSubnet := armnetwork.Subnet{
					ID: ptr.To(subnetIdWorker),
					Properties: &armnetwork.SubnetPropertiesFormat{
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:   ptr.To("Microsoft.ContainerRegistry"),
								Locations: []*string{ptr.To("*")},
							},
							{
								Service:   ptr.To("Microsoft.Storage"),
								Locations: []*string{ptr.To("*")},
							},
						},
					},
				}
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, gomock.Any(), nil).DoAndReturn(
					func(_, _, _, _ interface{}, subnet armnetwork.Subnet, _ interface{}) error {
						if !reflect.DeepEqual(subnet, expectedMasterSubnet) {
							t.Errorf("expected master subnet to be %v, got %v", expectedMasterSubnet, subnet)
						}
						return nil
					})
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, gomock.Any(), nil).DoAndReturn(
					func(_, _, _, _ interface{}, subnet armnetwork.Subnet, _ interface{}) error {
						if !reflect.DeepEqual(subnet, expectedWorkerSubnet) {
							t.Errorf("expected worker subnet to be %v, got %v", expectedWorkerSubnet, subnet)
						}
						return nil
					})
			},
		},
		{
			name: "It should return error when subnet ID is empty",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
					WorkerProfiles: []api.WorkerProfile{
						{
							Name:     "workerProfile",
							SubnetID: "",
						},
					},
				},
			},
			mock: func(subnets *mock_armnetwork.MockSubnetsClient) {
				subnets.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			expectedErr: "WorkerProfile 'workerProfile' has no SubnetID; check that the corresponding MachineSet is valid",
		},
		{
			name: "It should not update subnet when subnet already have service endpoints",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
					WorkerProfiles: []api.WorkerProfile{
						{SubnetID: subnetIdWorker},
					},
				},
			},
			mock: func(subnets *mock_armnetwork.MockSubnetsClient) {
				masterSubnetWithServiceEndpoints := masterSubnet
				masterSubnetWithServiceEndpoints.Properties.ServiceEndpoints = []*armnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:           ptr.To("Microsoft.Storage"),
						Locations:         []*string{ptr.To("*")},
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
					},
					{
						Service:           ptr.To("Microsoft.ContainerRegistry"),
						Locations:         []*string{ptr.To("*")},
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
					},
				}
				workerSubnetWithServiceEndpoints := workerSubnet
				workerSubnetWithServiceEndpoints.Properties.ServiceEndpoints = []*armnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:           ptr.To("Microsoft.Storage"),
						Locations:         []*string{ptr.To("*")},
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
					},
					{
						Service:           ptr.To("Microsoft.ContainerRegistry"),
						Locations:         []*string{ptr.To("*")},
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
					},
				}
				subnets.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: masterSubnetWithServiceEndpoints}, nil)
				subnets.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: workerSubnetWithServiceEndpoints}, nil)
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name: "It updates subnet when subnet already have one of the service endpoints",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
					WorkerProfiles: []api.WorkerProfile{
						{SubnetID: subnetIdWorker},
					},
				},
			},
			mock: func(subnets *mock_armnetwork.MockSubnetsClient) {
				masterSubnetWithServiceEndpoints := masterSubnet
				masterSubnetWithServiceEndpoints.Properties.ServiceEndpoints = []*armnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:           ptr.To("Microsoft.Storage"),
						Locations:         []*string{ptr.To("*")},
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
					},
				}
				workerSubnetWithServiceEndpoints := workerSubnet
				workerSubnetWithServiceEndpoints.Properties.ServiceEndpoints = []*armnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:           ptr.To("Microsoft.ContainerRegistry"),
						Locations:         []*string{ptr.To("*")},
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
					},
				}
				subnets.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: masterSubnetWithServiceEndpoints}, nil)
				subnets.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: workerSubnetWithServiceEndpoints}, nil)
				expectedMasterSubnet := armnetwork.Subnet{
					ID: ptr.To(subnetIdMaster),
					Properties: &armnetwork.SubnetPropertiesFormat{
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           ptr.To("Microsoft.Storage"),
								Locations:         []*string{ptr.To("*")},
								ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
							},
							{
								Service:   ptr.To("Microsoft.ContainerRegistry"),
								Locations: []*string{ptr.To("*")},
							},
						},
					},
				}
				expectedWorkerSubnet := armnetwork.Subnet{
					ID: ptr.To(subnetIdWorker),
					Properties: &armnetwork.SubnetPropertiesFormat{
						ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           ptr.To("Microsoft.ContainerRegistry"),
								Locations:         []*string{ptr.To("*")},
								ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
							},
							{
								Service:   ptr.To("Microsoft.Storage"),
								Locations: []*string{ptr.To("*")},
							},
						},
					},
				}
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, gomock.Any(), nil).DoAndReturn(
					func(_, _, _, _ interface{}, subnet armnetwork.Subnet, _ interface{}) error {
						if !reflect.DeepEqual(subnet, expectedMasterSubnet) {
							t.Errorf("expected master subnet to be %v, got %v", expectedMasterSubnet, subnet)
						}
						return nil
					})
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, gomock.Any(), nil).DoAndReturn(
					func(_, _, _, _ interface{}, subnet armnetwork.Subnet, _ interface{}) error {
						if !reflect.DeepEqual(subnet, expectedWorkerSubnet) {
							t.Errorf("expected worker subnet to be %v, got %v", expectedWorkerSubnet, subnet)
						}
						return nil
					})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx := context.Background()

			subnetsClient := mock_armnetwork.NewMockSubnetsClient(controller)
			tt.mock(subnetsClient)

			m := &manager{
				armSubnets: subnetsClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: tt.oc,
				},
			}

			err := m.ensureServiceEndpoints(ctx)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAddEndpointsToSubnets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		subnet         *armnetwork.Subnet
		newEndpoints   []string
		expectedSubnet *armnetwork.Subnet
		expectedResult bool
	}{
		{
			name: "addEndpointsToSubnet should  when the subnet contains all new endpoints in succeeded state",
			subnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
						{
							Service:           ptr.To("Microsoft.ContainerRegistry"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						},
						{
							Service:           ptr.To("Microsoft.Storage"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						},
					},
				},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
						{
							Service:           ptr.To("Microsoft.ContainerRegistry"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						},
						{
							Service:           ptr.To("Microsoft.Storage"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "addEndpointsToSubnet should update the subnet when it has no service endpoints",
			subnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{},
				},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
						{
							Service:   ptr.To("Microsoft.ContainerRegistry"),
							Locations: []*string{ptr.To("*")},
						},
						{
							Service:   ptr.To("Microsoft.Storage"),
							Locations: []*string{ptr.To("*")},
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "addEndpointsToSubnet should update the subnet when ServiceEndpoints is nil",
			subnet: &armnetwork.Subnet{
				ID:         ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
						{
							Service:   ptr.To("Microsoft.ContainerRegistry"),
							Locations: []*string{ptr.To("*")},
						},
						{
							Service:   ptr.To("Microsoft.Storage"),
							Locations: []*string{ptr.To("*")},
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "addEndpointsToSubnet should update the subnet (with 4 endpoints: 2 previous in failed state + 2 new) when it contains all new endpoints but those are not in succeeded state",
			subnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
						{
							Service:           ptr.To("Microsoft.ContainerRegistry"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateFailed),
						},
						{
							Service:           ptr.To("Microsoft.Storage"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateFailed),
						},
					},
				},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
						{
							Service:           ptr.To("Microsoft.ContainerRegistry"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateFailed),
						},
						{
							Service:           ptr.To("Microsoft.Storage"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateFailed),
						},
						{
							Service:   ptr.To("Microsoft.ContainerRegistry"),
							Locations: []*string{ptr.To("*")},
						},
						{
							Service:   ptr.To("Microsoft.Storage"),
							Locations: []*string{ptr.To("*")},
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "addEndpointsToSubnet should return an updated Subnet (with 2 endpoints: 1 previous was already in succeeded state + 1 new (it was missing))",
			subnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
						{
							Service:           ptr.To("Microsoft.ContainerRegistry"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						},
					},
				},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnet: &armnetwork.Subnet{
				ID: ptr.To(subnetIdMaster),
				Properties: &armnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
						{
							Service:           ptr.To("Microsoft.ContainerRegistry"),
							Locations:         []*string{ptr.To("*")},
							ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
						},
						{
							Service:   ptr.To("Microsoft.Storage"),
							Locations: []*string{ptr.To("*")},
						},
					},
				},
			},
			expectedResult: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := addEndpointsToSubnet(tt.newEndpoints, tt.subnet)

			assert.Equal(t, tt.expectedResult, result)
			if !reflect.DeepEqual(tt.expectedSubnet, tt.subnet) {
				t.Fatalf("subnet is different than expectedSubnet. Expected %v, but got %v", tt.expectedSubnet, tt.subnet)
			}
		})
	}
}
