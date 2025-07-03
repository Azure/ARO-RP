package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestUpdateLoadBalancerZonal(t *testing.T) {
	ctx := context.Background()
	infraID := "infraID"
	location := "eastus"
	rgName := "clusterRG"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/" + rgName
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	for _, tt := range []struct {
		name                string
		architectureVersion api.ArchitectureVersion
		mocks               func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_compute.MockResourceSkusClient)
		wantErr             error
		expectedLogs        []map[string]types.GomegaMatcher
	}{
		{
			name:                "noop -- has zones from creation",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_compute.MockResourceSkusClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							ID: pointerutils.ToPtr(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
										Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
									},
								},
							},
						},
					}, nil,
				)
			},
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("internal load balancer frontend IP already zone-redundant, no need to continue"),
				},
			},
		},
		{
			name:                "noop -- already performed",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_compute.MockResourceSkusClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							ID: pointerutils.ToPtr(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  pointerutils.ToPtr("internal-lb-ip-zonal-v4"),
										Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
									},
									{
										Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
										Zones: []*string{},
									},
								},
							},
						},
					}, nil,
				)
			},
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("zone-redundant frontend IP already exists, no need to continue"),
				},
			},
		},
		{
			name:                "noop -- non-zonal",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_compute.MockResourceSkusClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							ID: pointerutils.ToPtr(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
										Zones: []*string{},
									},
								},
							},
						},
					}, nil,
				)

				sku.EXPECT().List(gomock.Any(), "location eq eastus").Return([]mgmtcompute.ResourceSku{
					{
						Name:      pointerutils.ToPtr(string(api.VMSizeStandardD16asV4)),
						Locations: &[]string{"eastus"},
						LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
							{
								Zones: pointerutils.ToPtr([]string{}),
							},
						},
						Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
						ResourceType: pointerutils.ToPtr("virtualMachines"),
					},
				}, nil)
			},
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("non-zonal control plane SKU, not adding zone-redundant frontend IP"),
				},
			},
		},
		{
			name:                "noop -- broken VM SKU",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_compute.MockResourceSkusClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							ID: pointerutils.ToPtr(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
										Zones: []*string{},
									},
								},
							},
						},
					}, nil,
				)

				sku.EXPECT().List(gomock.Any(), "location eq eastus").Return([]mgmtcompute.ResourceSku{
					{
						Name:         pointerutils.ToPtr(string(api.VMSizeStandardD16asV4)),
						Locations:    &[]string{"eastus"},
						LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{},
						Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
						ResourceType: pointerutils.ToPtr("virtualMachines"),
					},
				}, nil)
			},
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("error determining the VM SKU availability, skipping: 400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_D16as_v4' is unavailable in region 'eastus'"),
				},
			},
		},
		{
			name:                "performed, zonal, v2",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_compute.MockResourceSkusClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							ID: pointerutils.ToPtr(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
										Zones: []*string{},
									},
								},
							},
						},
					}, nil,
				)

				sku.EXPECT().List(gomock.Any(), "location eq eastus").Return([]mgmtcompute.ResourceSku{
					{
						Name:      pointerutils.ToPtr(string(api.VMSizeStandardD16asV4)),
						Locations: &[]string{"eastus"},
						LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
							{Zones: &[]string{"1", "2", "3"}},
						},
						Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
						ResourceType: pointerutils.ToPtr("virtualMachines"),
					},
				}, nil)

				lbs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, infraID+"-internal",
					armnetwork.LoadBalancer{
						ID: pointerutils.ToPtr(infraID + "-internal"),
						Properties: &armnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
								{
									Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
									Zones: []*string{},
								},
								{
									Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
										PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
										Subnet: &armnetwork.Subnet{
											ID: pointerutils.ToPtr("subnetID"),
										},
									},
									Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
									Name:  pointerutils.ToPtr(zonalFrontendIPName),
								},
							},

							LoadBalancingRules: []*armnetwork.LoadBalancingRule{
								{
									Name: pointerutils.ToPtr("api-internal-v4-zonal"),
									Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal/frontendIPConfigurations/internal-lb-ip-zonal-v4"),
										},
										BackendAddressPool: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal/backendAddressPools/infraID"),
										},
										Probe: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal/probes/api-internal-probe"),
										},
										Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
										LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
										FrontendPort:         pointerutils.ToPtr(int32(6443)),
										BackendPort:          pointerutils.ToPtr(int32(6443)),
										IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
										DisableOutboundSnat:  pointerutils.ToPtr(true),
									},
								},
								{
									Name: pointerutils.ToPtr("sint-v4-zonal"),
									Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal/frontendIPConfigurations/internal-lb-ip-zonal-v4"),
										},
										BackendAddressPool: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal/backendAddressPools/infraID"),
										},
										Probe: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal/probes/sint-probe"),
										},
										Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
										LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
										FrontendPort:         pointerutils.ToPtr(int32(22623)),
										BackendPort:          pointerutils.ToPtr(int32(22623)),
										IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
									},
								},
							},
						},
					}, nil).Return(nil)
			},
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("updating internal load balancer with zone-redundant frontend IP"),
				},
			},
		},
		{
			name:                "performed, zonal, v1",
			architectureVersion: api.ArchitectureVersionV1,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_compute.MockResourceSkusClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal-lb", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							ID: pointerutils.ToPtr(infraID + "-internal-lb"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
										Zones: []*string{},
									},
								},
							},
						},
					}, nil,
				)

				sku.EXPECT().List(gomock.Any(), "location eq eastus").Return([]mgmtcompute.ResourceSku{
					{
						Name:      pointerutils.ToPtr(string(api.VMSizeStandardD16asV4)),
						Locations: &[]string{"eastus"},
						LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
							{Zones: &[]string{"1", "2", "3"}},
						},
						Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
						ResourceType: pointerutils.ToPtr("virtualMachines"),
					},
				}, nil)

				lbs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, infraID+"-internal-lb",
					armnetwork.LoadBalancer{
						ID: pointerutils.ToPtr(infraID + "-internal-lb"),
						Properties: &armnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
								{
									Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
									Zones: []*string{},
								},
								{
									Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
										PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
										Subnet: &armnetwork.Subnet{
											ID: pointerutils.ToPtr("subnetID"),
										},
									},
									Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
									Name:  pointerutils.ToPtr(zonalFrontendIPName),
								},
							},

							LoadBalancingRules: []*armnetwork.LoadBalancingRule{
								{
									Name: pointerutils.ToPtr("api-internal-v4-zonal"),
									Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal-lb/frontendIPConfigurations/internal-lb-ip-zonal-v4"),
										},
										BackendAddressPool: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal-lb/backendAddressPools/infraID"),
										},
										Probe: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal-lb/probes/api-internal-probe"),
										},
										Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
										LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
										FrontendPort:         pointerutils.ToPtr(int32(6443)),
										BackendPort:          pointerutils.ToPtr(int32(6443)),
										IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
										DisableOutboundSnat:  pointerutils.ToPtr(true),
									},
								},
								{
									Name: pointerutils.ToPtr("sint-v4-zonal"),
									Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal-lb/frontendIPConfigurations/internal-lb-ip-zonal-v4"),
										},
										BackendAddressPool: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal-lb/backendAddressPools/infraID"),
										},
										Probe: &armnetwork.SubResource{
											ID: pointerutils.ToPtr(infraID + "-internal-lb/probes/sint-probe"),
										},
										Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
										LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
										FrontendPort:         pointerutils.ToPtr(int32(22623)),
										BackendPort:          pointerutils.ToPtr(int32(22623)),
										IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
									},
								},
							},
						},
					}, nil).Return(nil)
			},
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("updating internal load balancer with zone-redundant frontend IP"),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			loadBalancers := mock_armnetwork.NewMockLoadBalancersClient(ctrl)
			skus := mock_compute.NewMockResourceSkusClient(ctrl)

			env := mock_env.NewMockInterface(ctrl)
			env.EXPECT().FeatureIsSet(gomock.Any()).AnyTimes().Return(false)

			hook, entry := testlog.New()

			doc := &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       key,
					Location: location,
					Properties: api.OpenShiftClusterProperties{
						ArchitectureVersion: tt.architectureVersion,
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: clusterRGID,
						},
						InfraID: infraID,
						NetworkProfile: api.NetworkProfile{
							LoadBalancerProfile: &api.LoadBalancerProfile{},
						},
						MasterProfile: api.MasterProfile{
							VMSize:   api.VMSizeStandardD16asV4,
							SubnetID: "subnetID",
						},
					},
				},
			}

			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(doc)
			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			tt.mocks(loadBalancers, skus)

			manager := manager{
				doc:              doc,
				db:               openShiftClustersDatabase,
				log:              entry,
				armLoadBalancers: loadBalancers,
				resourceSkus:     skus,
				env:              env,
			}

			err = manager.fixInternalLoadBalancerZones(ctx)
			assert.Equal(t, tt.wantErr, err)

			err = testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
