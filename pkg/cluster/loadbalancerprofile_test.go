package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/loadbalancer"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	uuidfake "github.com/Azure/ARO-RP/pkg/util/uuid/fake"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestReconcileOutboundIPs(t *testing.T) {
	ctx := context.Background()
	infraID := "infraID"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"
	clusterRGName := "clusterRG"
	location := "eastus"

	// Run tests
	for _, tt := range []struct {
		name  string
		m     manager
		uuids []string
		mocks func(
			publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
			ctx context.Context)
		expectedOutboundIPS []api.ResourceReference
		expectedErr         error
	}{
		{
			name: "create 1 additional managed ip",
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							NetworkProfile: api.NetworkProfile{
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 3,
									},
								},
							},
						},
					},
				},
			},
			uuids: []string{"uuid2"},
			mocks: func(
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid2-outbound-pip-v4", getFakePublicIPAddress("uuid2-outbound-pip-v4", location)).
					Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "no additional managed ip needed",
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							NetworkProfile: api.NetworkProfile{
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 2,
									},
								},
							},
						},
					},
				},
			},
			uuids: []string{},
			mocks: func(
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(2), nil)
			},
			expectedErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.log = logrus.NewEntry(logrus.StandardLogger())
			uuid.DefaultGenerator = uuidfake.NewGenerator(tt.uuids)
			controller := gomock.NewController(t)
			defer controller.Finish()
			publicIPAddressClient := mock_network.NewMockPublicIPAddressesClient(controller)

			if tt.mocks != nil {
				tt.mocks(publicIPAddressClient, ctx)
			}
			tt.m.publicIPAddresses = publicIPAddressClient

			// Run reconcileOutboundIPs and assert the correct results
			outboundIPs, err := tt.m.reconcileOutboundIPs(ctx)
			assert.Equal(t, tt.expectedErr, err, "Unexpected error exception")
			// results are not deterministic when scaling down so just check desired length
			assert.Len(t, outboundIPs, tt.m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count)
		})
	}
}

func TestDeleteUnusedManagedIPs(t *testing.T) {
	ctx := context.Background()
	infraID := "infraID"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"
	clusterRGName := "clusterRG"

	// Run tests
	for _, tt := range []struct {
		name  string
		m     manager
		mocks func(
			publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
			loadBalancersClient *mock_network.MockLoadBalancersClient,
			ctx context.Context)
		expectedManagedIPs map[string]mgmtnetwork.PublicIPAddress
		expectedErr        error
	}{
		{
			name: "delete unused managed IPs except api server ip",
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							NetworkProfile: api.NetworkProfile{
								LoadBalancerProfile: &api.LoadBalancerProfile{
									EffectiveOutboundIPs: []api.EffectiveOutboundIP{
										{
											ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
										},
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4")
			},
			expectedErr: nil,
		},
		{
			name: "delete unused managed IPs",
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPrivate,
							},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							NetworkProfile: api.NetworkProfile{
								LoadBalancerProfile: &api.LoadBalancerProfile{
									EffectiveOutboundIPs: []api.EffectiveOutboundIP{
										{
											ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/customerRG/providers/Microsoft.Network/publicIPAddress/ip",
										},
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(mgmtnetwork.LoadBalancer{
						Name: &infraID,
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
								{
									Name: to.StringPtr("ae3506385907e44eba9ef9bf76eac973"),
									ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &mgmtnetwork.PublicIPAddress{
											ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
										},
									},
								},
								{
									Name: to.StringPtr("customer-ip"),
									ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/customerRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/customer-ip"),
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &mgmtnetwork.PublicIPAddress{
											ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/customerRG/providers/Microsoft.Network/publicIPAddresses/customer-ip"),
										},
									},
								},
							},
							OutboundRules: &[]mgmtnetwork.OutboundRule{
								{
									Name: to.StringPtr(loadbalancer.OutboundRuleV4),
									OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
										FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
											{
												ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/customerRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/customer-ip"),
											},
										},
									},
								},
							},
						},
					}, nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "infraID-pip-v4")
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4")
			},
			expectedErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.log = logrus.NewEntry(logrus.StandardLogger())

			controller := gomock.NewController(t)
			defer controller.Finish()
			publicIPAddressClient := mock_network.NewMockPublicIPAddressesClient(controller)
			loadBalancersClient := mock_network.NewMockLoadBalancersClient(controller)

			if tt.mocks != nil {
				tt.mocks(publicIPAddressClient, loadBalancersClient, ctx)
			}
			tt.m.publicIPAddresses = publicIPAddressClient
			tt.m.loadBalancers = loadBalancersClient

			// Run deleteUnusedManagedIPs and assert the correct results
			err := tt.m.deleteUnusedManagedIPs(ctx)
			assert.Equal(t, tt.expectedErr, err, "Unexpected error exception")
		})
	}
}

func TestReconcileLoadBalancerProfile(t *testing.T) {
	ctx := context.Background()
	infraID := "infraID"
	location := "eastus"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"
	clusterRGName := "clusterRG"
	defaultOutboundIPName := infraID + "-pip-v4"
	defaultOutboundIPID := clusterRGID + "/providers/Microsoft.Network/publicIPAddresses/" + defaultOutboundIPName
	// Define the DB instance we will use to run the PatchWithLease function
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	// Run tests
	for _, tt := range []struct {
		name                        string
		m                           manager
		lb                          mgmtnetwork.LoadBalancer
		expectedLoadBalancerProfile *api.LoadBalancerProfile
		uuids                       []string
		mocks                       func(
			loadBalancersClient *mock_network.MockLoadBalancersClient,
			publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
			ctx context.Context)
		expectedErr []error
	}{
		{
			name:  "reconcile is skipped when outboundType is UserDefinedRouting",
			uuids: []string{},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: api.ArchitectureVersionV2,
							ProvisioningState:   api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType:        api.OutboundTypeUserDefinedRouting,
								LoadBalancerProfile: nil,
							},
						},
					},
				},
			},
			expectedLoadBalancerProfile: nil,
			expectedErr:                 nil,
		},
		{
			name:  "reconcile is skipped when architecture version is V1",
			uuids: []string{},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: api.ArchitectureVersionV1,
							ProvisioningState:   api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType:        api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: nil,
							},
						},
					},
				},
			},
			expectedLoadBalancerProfile: nil,
			expectedErr:                 nil,
		},
		{
			name:  "default managed ips",
			uuids: []string{},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: api.ArchitectureVersionV2,
							ProvisioningState:   api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(0), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(0), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
			},
			expectedLoadBalancerProfile: &api.LoadBalancerProfile{
				ManagedOutboundIPs: &api.ManagedOutboundIPs{
					Count: 1,
				},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{
					{
						ID: defaultOutboundIPID,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:  "effectiveOutboundIPs is patched when effectiveOutboundIPs does not match load balancer",
			uuids: []string{},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateUpdating,
							ArchitectureVersion: api.ArchitectureVersionV2,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 2,
									},
									EffectiveOutboundIPs: []api.EffectiveOutboundIP{
										{
											ID: defaultOutboundIPID,
										},
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(1, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(1, api.VisibilityPublic), nil)
			},
			expectedLoadBalancerProfile: &api.LoadBalancerProfile{
				ManagedOutboundIPs: &api.ManagedOutboundIPs{
					Count: 2,
				},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{
					{
						ID: defaultOutboundIPID,
					},
					{
						ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:  "add one IP to the default public load balancer",
			uuids: []string{"uuid1"},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: api.ArchitectureVersionV2,
							ProvisioningState:   api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 2,
									},
									EffectiveOutboundIPs: []api.EffectiveOutboundIP{
										{
											ID: defaultOutboundIPID,
										},
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(0), nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid1-outbound-pip-v4", getFakePublicIPAddress("uuid1-outbound-pip-v4", location)).Return(nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, loadbalancer.FakeUpdatedLoadBalancer(1)).Return(nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(1, api.VisibilityPublic), nil)
			},
			expectedLoadBalancerProfile: &api.LoadBalancerProfile{
				ManagedOutboundIPs: &api.ManagedOutboundIPs{
					Count: 2,
				},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{
					{

						ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
					},
					{
						ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:  "remove one IP from the default public load balancer",
			uuids: []string{},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateUpdating,
							ArchitectureVersion: api.ArchitectureVersionV2,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
									EffectiveOutboundIPs: []api.EffectiveOutboundIP{
										{
											ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
										},
										{
											ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
										},
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(1, api.VisibilityPublic), nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, loadbalancer.FakeUpdatedLoadBalancer(0)).Return(nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4")
			},

			expectedLoadBalancerProfile: &api.LoadBalancerProfile{
				ManagedOutboundIPs: &api.ManagedOutboundIPs{
					Count: 1,
				},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{
					{
						ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:  "created IPs cleaned up when update fails",
			uuids: []string{"uuid1"},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateUpdating,
							ArchitectureVersion: api.ArchitectureVersionV2,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 2,
									},
									EffectiveOutboundIPs: []api.EffectiveOutboundIP{
										{
											ID: defaultOutboundIPID,
										},
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(0), nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid1-outbound-pip-v4", getFakePublicIPAddress("uuid1-outbound-pip-v4", location)).Return(nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, loadbalancer.FakeUpdatedLoadBalancer(1)).Return(fmt.Errorf("lb update failed"))
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4")
			},
			expectedLoadBalancerProfile: &api.LoadBalancerProfile{
				ManagedOutboundIPs: &api.ManagedOutboundIPs{
					Count: 2,
				},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{
					{
						ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
					},
				},
			},
			expectedErr: []error{fmt.Errorf("lb update failed")},
		},
		{
			name:  "managed ip cleanup errors are propagated when cleanup fails",
			uuids: []string{"uuid1"},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateUpdating,
							ArchitectureVersion: api.ArchitectureVersionV2,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
									EffectiveOutboundIPs: []api.EffectiveOutboundIP{
										{
											ID: defaultOutboundIPID,
										},
										{
											ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
										},
										{
											ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid2-outbound-pip-v4",
										},
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(2), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(2, api.VisibilityPublic), nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, loadbalancer.FakeUpdatedLoadBalancer(0)).Return(nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(2), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4").Return(fmt.Errorf("error"))
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid2-outbound-pip-v4").Return(fmt.Errorf("error"))
			},
			expectedLoadBalancerProfile: &api.LoadBalancerProfile{
				ManagedOutboundIPs: &api.ManagedOutboundIPs{
					Count: 1,
				},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{
					{
						ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
					},
				},
			},
			expectedErr: []error{fmt.Errorf("failed to cleanup unused managed ips\ndeletion of unused managed ip uuid1-outbound-pip-v4 failed with error: error\ndeletion of unused managed ip uuid2-outbound-pip-v4 failed with error: error"), fmt.Errorf("failed to cleanup unused managed ips\ndeletion of unused managed ip uuid2-outbound-pip-v4 failed with error: error\ndeletion of unused managed ip uuid1-outbound-pip-v4 failed with error: error")},
		},
		{
			name:  "all errors propagated",
			uuids: []string{"uuid1", "uuid2"},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateUpdating,
							ArchitectureVersion: api.ArchitectureVersionV2,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 3,
									},
									EffectiveOutboundIPs: []api.EffectiveOutboundIP{
										{
											ID: defaultOutboundIPID,
										},
									},
								},
							},
						},
					},
				},
			},
			mocks: func(
				loadBalancersClient *mock_network.MockLoadBalancersClient,
				publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(0), nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid1-outbound-pip-v4", getFakePublicIPAddress("uuid1-outbound-pip-v4", location)).Return(nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid2-outbound-pip-v4", getFakePublicIPAddress("uuid2-outbound-pip-v4", location)).Return(fmt.Errorf("failed to create ip"))
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(loadbalancer.FakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4").Return(fmt.Errorf("error"))
			},
			expectedLoadBalancerProfile: &api.LoadBalancerProfile{
				ManagedOutboundIPs: &api.ManagedOutboundIPs{
					Count: 3,
				},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{
					{
						ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
					},
				},
			},
			expectedErr: []error{fmt.Errorf("multiple errors occurred while updating outbound-rule-v4\nfailed to create required IPs\ncreation of ip address uuid2-outbound-pip-v4 failed with error: failed to create ip\nfailed to cleanup unused managed ips\ndeletion of unused managed ip uuid1-outbound-pip-v4 failed with error: error")},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Create the DB to test the cluster
			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(tt.m.doc)
			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}
			tt.m.db = openShiftClustersDatabase
			tt.m.log = logrus.NewEntry(logrus.StandardLogger())

			uuid.DefaultGenerator = uuidfake.NewGenerator(tt.uuids)
			controller := gomock.NewController(t)
			defer controller.Finish()
			loadBalancersClient := mock_network.NewMockLoadBalancersClient(controller)
			publicIPAddressClient := mock_network.NewMockPublicIPAddressesClient(controller)

			if tt.mocks != nil {
				tt.mocks(loadBalancersClient, publicIPAddressClient, ctx)
			}
			tt.m.loadBalancers = loadBalancersClient
			tt.m.publicIPAddresses = publicIPAddressClient

			// Run reconcileLoadBalancerProfile and assert the correct results
			err = tt.m.reconcileLoadBalancerProfile(ctx)
			// Expect error to be in the array of errors provided or to be nil
			if tt.expectedErr != nil {
				assert.Contains(t, tt.expectedErr, err, "Unexpected error exception")
			} else {
				require.NoError(t, err, "Unexpected error exception")
			}
			assert.Equal(t, &tt.expectedLoadBalancerProfile, &tt.m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile)
		})
	}
}

func getFakePublicIPAddress(name, location string) mgmtnetwork.PublicIPAddress {
	id := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/%s", name)
	return mgmtnetwork.PublicIPAddress{
		Name:     &name,
		ID:       &id,
		Location: to.StringPtr(location),
		PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: mgmtnetwork.Static,
			PublicIPAddressVersion:   mgmtnetwork.IPv4,
		},
		Sku: &mgmtnetwork.PublicIPAddressSku{
			Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
		},
	}
}

func getFakePublicIPList(managedCount int) []mgmtnetwork.PublicIPAddress {
	infraID := "infraID"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"
	defaultOutboundIPName := infraID + "-pip-v4"
	defaultOutboundIPID := clusterRGID + "/providers/Microsoft.Network/publicIPAddresses/" + defaultOutboundIPName
	ips := []mgmtnetwork.PublicIPAddress{
		{
			ID:   &defaultOutboundIPID,
			Name: &defaultOutboundIPName,
		},
		{
			ID:   to.StringPtr(clusterRGID + "/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
			Name: to.StringPtr("infraID-default-v4"),
		},
	}
	for i := 0; i < managedCount; i++ {
		ipName := fmt.Sprintf("uuid%d-outbound-pip-v4", i+1)
		ips = append(ips, getFakePublicIPAddress(ipName, "eastus"))
	}
	return ips
}
