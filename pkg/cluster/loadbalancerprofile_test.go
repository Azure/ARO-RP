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

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	uuidfake "github.com/Azure/ARO-RP/pkg/util/uuid/fake"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestGetDesiredOutboundIPs(t *testing.T) {
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

			// Run getDesiredOutboundIPs and assert the correct results
			outboundIPs, err := tt.m.getDesiredOutboundIPs(ctx)
			assert.Equal(t, tt.expectedErr, err, "Unexpected error exception")
			// results are not deterministic when scaling down so just check desired length
			assert.Equal(t, tt.m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count, len(outboundIPs))
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
					Return(fakeLoadBalancersGet(0, api.VisibilityPublic), nil)
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
									Name: to.StringPtr(outboundRuleV4),
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

func TestAddOutboundIPsToLB(t *testing.T) {
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"

	// Run tests
	for _, tt := range []struct {
		name         string
		desiredOBIPs []api.ResourceReference
		currentLB    mgmtnetwork.LoadBalancer
		expectedLB   mgmtnetwork.LoadBalancer
	}{
		{
			name: "add default IP to lb",
			desiredOBIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
			},
			currentLB:  getClearedLB(),
			expectedLB: fakeUpdatedLoadBalancer(0),
		},
		{
			name: "add multiple outbound IPs to LB",
			desiredOBIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
				},
			},
			currentLB:  getClearedLB(),
			expectedLB: fakeUpdatedLoadBalancer(1),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Run addOutboundIPsToLB and assert the correct results
			addOutboundIPsToLB(clusterRGID, tt.currentLB, tt.desiredOBIPs)
			assert.Equal(t, tt.expectedLB, tt.currentLB)
		})
	}
}

func TestRemoveOutboundIPsFromLB(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name       string
		currentLB  mgmtnetwork.LoadBalancer
		expectedLB mgmtnetwork.LoadBalancer
	}{
		{
			name:      "remove all outbound-rule-v4 fip config except api server",
			currentLB: fakeLoadBalancersGet(1, api.VisibilityPublic),
			expectedLB: mgmtnetwork.LoadBalancer{
				Name: &infraID,
				LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
						{
							Name: to.StringPtr("ae3506385907e44eba9ef9bf76eac973"),
							ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
							FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
									},
									{
										ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
									},
								},
								PublicIPAddress: &mgmtnetwork.PublicIPAddress{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
								},
							},
						},
						{
							Name: to.StringPtr("public-lb-ip-v4"),
							ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("api-internal-v4"),
									},
								},
								OutboundRules: &[]mgmtnetwork.SubResource{{
									ID: to.StringPtr(outboundRuleV4),
								}},
								PublicIPAddress: &mgmtnetwork.PublicIPAddress{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
								},
							},
						},
					},
					OutboundRules: &[]mgmtnetwork.OutboundRule{
						{
							Name: to.StringPtr(outboundRuleV4),
							OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: &[]mgmtnetwork.SubResource{},
							},
						},
					},
				},
			},
		},
		{
			name:      "remove all outbound-rule-v4 fip config",
			currentLB: fakeLoadBalancersGet(1, api.VisibilityPrivate),
			expectedLB: mgmtnetwork.LoadBalancer{
				Name: &infraID,
				LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
						{
							Name: to.StringPtr("ae3506385907e44eba9ef9bf76eac973"),
							ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
							FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
									},
									{
										ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
									},
								},
								PublicIPAddress: &mgmtnetwork.PublicIPAddress{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
								},
							},
						},
					},
					OutboundRules: &[]mgmtnetwork.OutboundRule{
						{
							Name: to.StringPtr(outboundRuleV4),
							OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: &[]mgmtnetwork.SubResource{},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Run removeOutboundIPsFromLB and assert correct results
			removeOutboundIPsFromLB(tt.currentLB)
			assert.Equal(t, tt.expectedLB, tt.currentLB)
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
		expectedLoadBalancerProfile api.LoadBalancerProfile
		uuids                       []string
		mocks                       func(
			loadBalancersClient *mock_network.MockLoadBalancersClient,
			publicIPAddressClient *mock_network.MockPublicIPAddressesClient,
			ctx context.Context)
		expectedErr error
	}{
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
							ProvisioningState: api.ProvisioningStateUpdating,
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
					Return(fakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(0), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(fakeLoadBalancersGet(0, api.VisibilityPublic), nil)
			},
			expectedLoadBalancerProfile: api.LoadBalancerProfile{
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
							ProvisioningState: api.ProvisioningStateUpdating,
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
					Return(fakeLoadBalancersGet(1, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(fakeLoadBalancersGet(1, api.VisibilityPublic), nil)
			},
			expectedLoadBalancerProfile: api.LoadBalancerProfile{
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
							ProvisioningState: api.ProvisioningStateUpdating,
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
					Return(fakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, fakeUpdatedLoadBalancer(1)).Return(nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(fakeLoadBalancersGet(1, api.VisibilityPublic), nil)
			},
			expectedLoadBalancerProfile: api.LoadBalancerProfile{
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
							ProvisioningState: api.ProvisioningStateUpdating,
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
					Return(fakeLoadBalancersGet(1, api.VisibilityPublic), nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, fakeUpdatedLoadBalancer(0)).Return(nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(fakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4")
			},

			expectedLoadBalancerProfile: api.LoadBalancerProfile{
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
							ProvisioningState: api.ProvisioningStateUpdating,
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
					Return(fakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, fakeUpdatedLoadBalancer(1)).Return(fmt.Errorf("lb update failed"))
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, "").
					Return(fakeLoadBalancersGet(0, api.VisibilityPublic), nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4")
			},
			expectedLoadBalancerProfile: api.LoadBalancerProfile{
				ManagedOutboundIPs: &api.ManagedOutboundIPs{
					Count: 2,
				},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{
					{
						ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
					},
				},
			},
			expectedErr: fmt.Errorf("lb update failed"),
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
			assert.Equal(t, tt.expectedErr, err, "Unexpected error exception")
			assert.Equal(t, &tt.expectedLoadBalancerProfile, tt.m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile)
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

// Returns a load balancer with config updated with desired outbound ips as it should be when m.loadBalancersClient.CreateOrUpdate is called.
// It is assumed that desired IPs include the default outbound IPs, however this won't work for transitions from
// customer provided IPs/Prefixes to managed IPs if the api server is private since the default IP
// would be deleted
func fakeUpdatedLoadBalancer(additionalIPCount int) mgmtnetwork.LoadBalancer {
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"
	defaultOutboundIPID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"
	lb := getClearedLB()
	ipResourceRefs := []api.ResourceReference{}
	ipResourceRefs = append(ipResourceRefs, api.ResourceReference{ID: defaultOutboundIPID})
	for i := 0; i < additionalIPCount; i++ {
		ipResourceRefs = append(ipResourceRefs, api.ResourceReference{ID: fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid%d-outbound-pip-v4", i+1)})
	}
	addOutboundIPsToLB(clusterRGID, lb, ipResourceRefs)
	return lb
}

// Returns lb as it would be returned via m.loadBalancersClient.Get.
func fakeLoadBalancersGet(additionalIPCount int, apiServerVisibility api.Visibility) mgmtnetwork.LoadBalancer {
	defaultOutboundFIPConfig := mgmtnetwork.FrontendIPConfiguration{
		Name: to.StringPtr("public-lb-ip-v4"),
		ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
		FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
			OutboundRules: &[]mgmtnetwork.SubResource{{
				ID: to.StringPtr(outboundRuleV4),
			}},
			PublicIPAddress: &mgmtnetwork.PublicIPAddress{
				ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
			},
		},
	}
	if apiServerVisibility == api.VisibilityPublic {
		defaultOutboundFIPConfig.FrontendIPConfigurationPropertiesFormat.LoadBalancingRules = &[]mgmtnetwork.SubResource{
			{
				ID: to.StringPtr("api-internal-v4"),
			},
		}
	}
	lb := mgmtnetwork.LoadBalancer{
		Name: &infraID,
		LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
				{
					Name: to.StringPtr("ae3506385907e44eba9ef9bf76eac973"),
					ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
					FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
						LoadBalancingRules: &[]mgmtnetwork.SubResource{
							{
								ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
							},
							{
								ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
							},
						},
						PublicIPAddress: &mgmtnetwork.PublicIPAddress{
							ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
						},
					},
				},
				defaultOutboundFIPConfig,
			},
			OutboundRules: &[]mgmtnetwork.OutboundRule{
				{
					Name: to.StringPtr(outboundRuleV4),
					OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
							{
								ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							},
						},
					},
				},
			},
		},
	}
	for i := 0; i < additionalIPCount; i++ {
		fipName := fmt.Sprintf("uuid%d-outbound-pip-v4", i+1)
		ipID := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid%d-outbound-pip-v4", i+1)
		fipID := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid%d-outbound-pip-v4", i+1)
		fipConfig := mgmtnetwork.FrontendIPConfiguration{
			Name: &fipName,
			ID:   &fipID,
			FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
				OutboundRules: &[]mgmtnetwork.SubResource{{
					ID: to.StringPtr(outboundRuleV4),
				}},
				PublicIPAddress: &mgmtnetwork.PublicIPAddress{
					ID: &ipID,
				},
			},
		}
		*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = append(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations, fipConfig)
		outboundRules := *lb.LoadBalancerPropertiesFormat.OutboundRules
		*outboundRules[0].FrontendIPConfigurations = append(*outboundRules[0].FrontendIPConfigurations, mgmtnetwork.SubResource{ID: fipConfig.ID})
	}
	return lb
}

func getClearedLB() mgmtnetwork.LoadBalancer {
	lb := fakeLoadBalancersGet(0, api.VisibilityPublic)
	removeOutboundIPsFromLB(lb)
	return lb
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
