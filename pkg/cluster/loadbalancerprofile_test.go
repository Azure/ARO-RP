package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"k8s.io/utils/ptr"

	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
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
			publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
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
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid2-outbound-pip-v4", getFakePublicIPAddress("uuid2-outbound-pip-v4", location), nil).
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
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
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
			publicIPAddressClient := mock_armnetwork.NewMockPublicIPAddressesClient(controller)

			if tt.mocks != nil {
				tt.mocks(publicIPAddressClient, ctx)
			}
			tt.m.armPublicIPAddresses = publicIPAddressClient

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
			publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
			loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
			ctx context.Context)
		expectedManagedIPs map[string]sdknetwork.PublicIPAddress
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
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4", nil)
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
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{
						LoadBalancer: sdknetwork.LoadBalancer{
							Name: &infraID,
							Properties: &sdknetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*sdknetwork.FrontendIPConfiguration{
									{
										Name: ptr.To("ae3506385907e44eba9ef9bf76eac973"),
										ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
										Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
											PublicIPAddress: &sdknetwork.PublicIPAddress{
												ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
											},
										},
									},
									{
										Name: ptr.To("customer-ip"),
										ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/customerRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/customer-ip"),
										Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
											PublicIPAddress: &sdknetwork.PublicIPAddress{
												ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/customerRG/providers/Microsoft.Network/publicIPAddresses/customer-ip"),
											},
										},
									},
								},
								OutboundRules: []*sdknetwork.OutboundRule{
									{
										Name: ptr.To(outboundRuleV4),
										Properties: &sdknetwork.OutboundRulePropertiesFormat{
											FrontendIPConfigurations: []*sdknetwork.SubResource{
												{
													ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/customerRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/customer-ip"),
												},
											},
										},
									},
								},
							},
						},
					}, nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "infraID-pip-v4", nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4", nil)
			},
			expectedErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.log = logrus.NewEntry(logrus.StandardLogger())

			controller := gomock.NewController(t)
			defer controller.Finish()
			publicIPAddressClient := mock_armnetwork.NewMockPublicIPAddressesClient(controller)
			loadBalancersClient := mock_armnetwork.NewMockLoadBalancersClient(controller)

			if tt.mocks != nil {
				tt.mocks(publicIPAddressClient, loadBalancersClient, ctx)
			}
			tt.m.armPublicIPAddresses = publicIPAddressClient
			tt.m.armLoadBalancers = loadBalancersClient

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
		currentLB    sdknetwork.LoadBalancer
		expectedLB   sdknetwork.LoadBalancer
	}{
		{
			name: "add default IP to lb",
			desiredOBIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
			},
			currentLB: getClearedLB(),
			expectedLB: sdknetwork.LoadBalancer{
				Name: ptr.To("infraID"),
				Properties: &sdknetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: []*sdknetwork.FrontendIPConfiguration{
						{
							Name: ptr.To("ae3506385907e44eba9ef9bf76eac973"),
							ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
							Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: []*sdknetwork.SubResource{
									{
										ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
									},
									{
										ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
									},
								},
								PublicIPAddress: &sdknetwork.PublicIPAddress{
									ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
								},
							},
						},
						{
							Name: ptr.To("public-lb-ip-v4"),
							ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: []*sdknetwork.SubResource{
									{
										ID: ptr.To("api-internal-v4"),
									},
								},
								OutboundRules: []*sdknetwork.SubResource{{
									ID: ptr.To(outboundRuleV4),
								}},
								PublicIPAddress: &sdknetwork.PublicIPAddress{
									ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
								},
							},
						},
					},
					OutboundRules: []*sdknetwork.OutboundRule{
						{
							Name: ptr.To(outboundRuleV4),
							Properties: &sdknetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: []*sdknetwork.SubResource{
									{
										ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
									},
								},
							},
						},
					},
				},
			},
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
			currentLB: getClearedLB(),
			expectedLB: sdknetwork.LoadBalancer{
				Name: ptr.To("infraID"),
				Properties: &sdknetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: []*sdknetwork.FrontendIPConfiguration{
						{
							Name: ptr.To("ae3506385907e44eba9ef9bf76eac973"),
							ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
							Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: []*sdknetwork.SubResource{
									{
										ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
									},
									{
										ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
									},
								},
								PublicIPAddress: &sdknetwork.PublicIPAddress{
									ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
								},
							},
						},
						{
							Name: ptr.To("public-lb-ip-v4"),
							ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: []*sdknetwork.SubResource{
									{
										ID: ptr.To("api-internal-v4"),
									},
								},
								OutboundRules: []*sdknetwork.SubResource{{
									ID: ptr.To(outboundRuleV4),
								}},
								PublicIPAddress: &sdknetwork.PublicIPAddress{
									ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
								},
							},
						},
						{
							Name: ptr.To("uuid1-outbound-pip-v4"),
							ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
							Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
								PublicIPAddress: &sdknetwork.PublicIPAddress{
									ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4"),
								},
							},
						},
					},
					OutboundRules: []*sdknetwork.OutboundRule{
						{
							Name: ptr.To(outboundRuleV4),
							Properties: &sdknetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: []*sdknetwork.SubResource{
									{
										ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
									},
									{
										ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
									},
								},
							},
						},
					},
				},
			},
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
		currentLB  sdknetwork.LoadBalancer
		expectedLB sdknetwork.LoadBalancer
	}{
		{
			name:      "remove all outbound-rule-v4 fip config except api server",
			currentLB: fakeLoadBalancersGet(1, api.VisibilityPublic),
			expectedLB: sdknetwork.LoadBalancer{
				Name: ptr.To("infraID"),
				Properties: &sdknetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: []*sdknetwork.FrontendIPConfiguration{
						{
							Name: ptr.To("ae3506385907e44eba9ef9bf76eac973"),
							ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
							Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: []*sdknetwork.SubResource{
									{
										ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
									},
									{
										ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
									},
								},
								PublicIPAddress: &sdknetwork.PublicIPAddress{
									ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
								},
							},
						},
						{
							Name: ptr.To("public-lb-ip-v4"),
							ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: []*sdknetwork.SubResource{
									{
										ID: ptr.To("api-internal-v4"),
									},
								},
								OutboundRules: []*sdknetwork.SubResource{{
									ID: ptr.To(outboundRuleV4),
								}},
								PublicIPAddress: &sdknetwork.PublicIPAddress{
									ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
								},
							},
						},
					},
					OutboundRules: []*sdknetwork.OutboundRule{
						{
							Name: ptr.To(outboundRuleV4),
							Properties: &sdknetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: []*sdknetwork.SubResource{},
							},
						},
					},
				},
			},
		},
		{
			name:      "remove all outbound-rule-v4 fip config",
			currentLB: fakeLoadBalancersGet(1, api.VisibilityPrivate),
			expectedLB: sdknetwork.LoadBalancer{
				Name: ptr.To("infraID"),
				Properties: &sdknetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: []*sdknetwork.FrontendIPConfiguration{
						{
							Name: ptr.To("ae3506385907e44eba9ef9bf76eac973"),
							ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
							Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: []*sdknetwork.SubResource{
									{
										ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
									},
									{
										ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
									},
								},
								PublicIPAddress: &sdknetwork.PublicIPAddress{
									ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
								},
							},
						},
					},
					OutboundRules: []*sdknetwork.OutboundRule{
						{
							Name: ptr.To(outboundRuleV4),
							Properties: &sdknetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: []*sdknetwork.SubResource{},
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
		lb                          sdknetwork.LoadBalancer
		expectedLoadBalancerProfile *api.LoadBalancerProfile
		uuids                       []string
		mocks                       func(
			loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
			publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
			ctx context.Context)
		expectedErr []error
	}{
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
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(0), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(0), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
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
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(1, api.VisibilityPublic)}, nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(1, api.VisibilityPublic)}, nil)
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
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(0), nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid1-outbound-pip-v4", getFakePublicIPAddress("uuid1-outbound-pip-v4", location), nil).Return(nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, fakeUpdatedLoadBalancer(1), nil).Return(nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(1, api.VisibilityPublic)}, nil)
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
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(1, api.VisibilityPublic)}, nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, fakeUpdatedLoadBalancer(0), nil).Return(nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4", nil)
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
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(0), nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid1-outbound-pip-v4", getFakePublicIPAddress("uuid1-outbound-pip-v4", location), nil).Return(nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, fakeUpdatedLoadBalancer(1), nil).Return(fmt.Errorf("lb update failed"))
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4", nil)
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
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(2), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(2, api.VisibilityPublic)}, nil)
				loadBalancersClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, infraID, fakeUpdatedLoadBalancer(0), nil).Return(nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(2), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4", nil).Return(fmt.Errorf("error"))
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid2-outbound-pip-v4", nil).Return(fmt.Errorf("error"))
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
				loadBalancersClient *mock_armnetwork.MockLoadBalancersClient,
				publicIPAddressClient *mock_armnetwork.MockPublicIPAddressesClient,
				ctx context.Context) {
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(0), nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid1-outbound-pip-v4", getFakePublicIPAddress("uuid1-outbound-pip-v4", location), nil).Return(nil)
				publicIPAddressClient.EXPECT().
					CreateOrUpdateAndWait(ctx, clusterRGName, "uuid2-outbound-pip-v4", getFakePublicIPAddress("uuid2-outbound-pip-v4", location), nil).Return(fmt.Errorf("failed to create ip"))
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				publicIPAddressClient.EXPECT().
					List(gomock.Any(), clusterRGName, nil).
					Return(getFakePublicIPList(1), nil)
				loadBalancersClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, nil).
					Return(sdknetwork.LoadBalancersClientGetResponse{LoadBalancer: fakeLoadBalancersGet(0, api.VisibilityPublic)}, nil)
				publicIPAddressClient.EXPECT().DeleteAndWait(gomock.Any(), "clusterRG", "uuid1-outbound-pip-v4", nil).Return(fmt.Errorf("error"))
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
			loadBalancersClient := mock_armnetwork.NewMockLoadBalancersClient(controller)
			publicIPAddressClient := mock_armnetwork.NewMockPublicIPAddressesClient(controller)

			if tt.mocks != nil {
				tt.mocks(loadBalancersClient, publicIPAddressClient, ctx)
			}
			tt.m.armLoadBalancers = loadBalancersClient
			tt.m.armPublicIPAddresses = publicIPAddressClient

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

func getFakePublicIPAddress(name, location string) sdknetwork.PublicIPAddress {
	id := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/%s", name)
	return sdknetwork.PublicIPAddress{
		Name:     &name,
		ID:       &id,
		Location: ptr.To(location),
		Properties: &sdknetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: ptr.To(sdknetwork.IPAllocationMethodStatic),
			PublicIPAddressVersion:   ptr.To(sdknetwork.IPVersionIPv4),
		},
		SKU: &sdknetwork.PublicIPAddressSKU{
			Name: ptr.To(sdknetwork.PublicIPAddressSKUNameStandard),
		},
	}
}

// Returns a load balancer with config updated with desired outbound ips as it should be when m.loadBalancersClient.CreateOrUpdate is called.
// It is assumed that desired IPs include the default outbound IPs, however this won't work for transitions from
// customer provided IPs/Prefixes to managed IPs if the api server is private since the default IP
// would be deleted
func fakeUpdatedLoadBalancer(additionalIPCount int) sdknetwork.LoadBalancer {
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
func fakeLoadBalancersGet(additionalIPCount int, apiServerVisibility api.Visibility) sdknetwork.LoadBalancer {
	defaultOutboundFIPConfig := sdknetwork.FrontendIPConfiguration{
		Name: ptr.To("public-lb-ip-v4"),
		ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
		Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
			OutboundRules: []*sdknetwork.SubResource{{
				ID: ptr.To(outboundRuleV4),
			}},
			PublicIPAddress: &sdknetwork.PublicIPAddress{
				ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
			},
		},
	}
	if apiServerVisibility == api.VisibilityPublic {
		defaultOutboundFIPConfig.Properties.LoadBalancingRules = []*sdknetwork.SubResource{
			{
				ID: ptr.To("api-internal-v4"),
			},
		}
	}
	lb := sdknetwork.LoadBalancer{
		Name: ptr.To("infraID"),
		Properties: &sdknetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*sdknetwork.FrontendIPConfiguration{
				{
					Name: ptr.To("ae3506385907e44eba9ef9bf76eac973"),
					ID:   ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
					Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
						LoadBalancingRules: []*sdknetwork.SubResource{
							{
								ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
							},
							{
								ID: ptr.To("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
							},
						},
						PublicIPAddress: &sdknetwork.PublicIPAddress{
							ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
						},
					},
				},
				&defaultOutboundFIPConfig,
			},
			OutboundRules: []*sdknetwork.OutboundRule{
				{
					Name: ptr.To(outboundRuleV4),
					Properties: &sdknetwork.OutboundRulePropertiesFormat{
						FrontendIPConfigurations: []*sdknetwork.SubResource{
							{
								ID: ptr.To("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
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
		fipConfig := &sdknetwork.FrontendIPConfiguration{
			Name: &fipName,
			ID:   &fipID,
			Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
				OutboundRules: []*sdknetwork.SubResource{{
					ID: ptr.To(outboundRuleV4),
				}},
				PublicIPAddress: &sdknetwork.PublicIPAddress{
					ID: &ipID,
				},
			},
		}
		lb.Properties.FrontendIPConfigurations = append(lb.Properties.FrontendIPConfigurations, fipConfig)
		outboundRules := lb.Properties.OutboundRules
		outboundRules[0].Properties.FrontendIPConfigurations = append(outboundRules[0].Properties.FrontendIPConfigurations, &sdknetwork.SubResource{ID: fipConfig.ID})
	}
	return lb
}

func getClearedLB() sdknetwork.LoadBalancer {
	lb := fakeLoadBalancersGet(0, api.VisibilityPublic)
	removeOutboundIPsFromLB(lb)
	return lb
}

func getFakePublicIPList(managedCount int) []*sdknetwork.PublicIPAddress {
	infraID := "infraID"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"
	defaultOutboundIPName := infraID + "-pip-v4"
	defaultOutboundIPID := clusterRGID + "/providers/Microsoft.Network/publicIPAddresses/" + defaultOutboundIPName
	ips := []*sdknetwork.PublicIPAddress{
		{
			ID:   &defaultOutboundIPID,
			Name: &defaultOutboundIPName,
		},
		{
			ID:   ptr.To(clusterRGID + "/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
			Name: ptr.To("infraID-default-v4"),
		},
	}
	for i := 0; i < managedCount; i++ {
		ipName := fmt.Sprintf("uuid%d-outbound-pip-v4", i+1)
		ip := getFakePublicIPAddress(ipName, "eastus")
		ips = append(ips, &ip)
	}
	return ips
}
