package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var errTestSKUFetchError = errors.New("oops")

func TestUpdateLoadBalancerZonalNoopAndErrorPaths(t *testing.T) {
	ctx := context.Background()
	infraID := "infraID"
	location := "eastus"
	rgName := "clusterRG"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/" + rgName
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	for _, tt := range []struct {
		name                string
		architectureVersion api.ArchitectureVersion
		mocks               func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_armcompute.MockResourceSKUsClient, plses *mock_armnetwork.MockPrivateLinkServicesClient)
		wantErrs            []error
		expectedLogs        []testlog.ExpectedLogEntry
	}{
		{
			name:                "noop -- already zone redundant",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_armcompute.MockResourceSKUsClient, plses *mock_armnetwork.MockPrivateLinkServicesClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							Name: new(infraID + "-internal"),
							ID:   new(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  new(internalLBFrontendIPName),
										Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
									},
								},
							},
						},
					}, nil,
				)
			},
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("internal load balancer frontend IP already zone-redundant, no need to continue"),
				},
			},
		},
		{
			name:                "noop -- non-zonal",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_armcompute.MockResourceSKUsClient, plses *mock_armnetwork.MockPrivateLinkServicesClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							Name: new(infraID + "-internal"),
							ID:   new(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  new("internal-lb-ip-v4"),
										Zones: []*string{},
									},
								},
							},
						},
					}, nil,
				)

				sku.EXPECT().List(gomock.Any(), "location eq eastus", false).Return([]*armcompute.ResourceSKU{
					{
						Name:      pointerutils.ToPtr(string(api.VMSizeStandardD16asV4)),
						Locations: pointerutils.ToSlicePtr([]string{"eastus"}),
						LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
							{
								Zones: []*string{},
							},
						}),
						Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
						ResourceType: new("virtualMachines"),
					},
				}, nil)
			},
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("non-zonal control plane SKU, not adding zone-redundant frontend IP"),
				},
			},
		},
		{
			name:                "noop -- missing VM SKU",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_armcompute.MockResourceSKUsClient, plses *mock_armnetwork.MockPrivateLinkServicesClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							ID:   new(infraID + "-internal"),
							Name: new(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  new("internal-lb-ip-v4"),
										Zones: []*string{},
									},
								},
							},
						},
					}, nil,
				)

				sku.EXPECT().List(gomock.Any(), "location eq eastus", false).Return([]*armcompute.ResourceSKU{
					{
						Name:         pointerutils.ToPtr(string(api.VMSizeStandardD16asV4)),
						Locations:    pointerutils.ToSlicePtr([]string{"eastus"}),
						LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{}),
						Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
						ResourceType: new("virtualMachines"),
					},
				}, nil)
			},
			expectedLogs: []testlog.ExpectedLogEntry{},
			wantErrs:     []error{errVMAvailability},
		},
		{
			name:                "noop -- error fetching SKU",
			architectureVersion: api.ArchitectureVersionV2,
			mocks: func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_armcompute.MockResourceSKUsClient, plses *mock_armnetwork.MockPrivateLinkServicesClient) {
				lbs.EXPECT().Get(gomock.Any(), rgName, infraID+"-internal", nil).Return(
					armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							ID:   new(infraID + "-internal"),
							Name: new(infraID + "-internal"),
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name:  new("internal-lb-ip-v4"),
										Zones: []*string{},
									},
								},
							},
						},
					}, nil,
				)

				sku.EXPECT().List(gomock.Any(), "location eq eastus", false).Return([]*armcompute.ResourceSKU{}, errTestSKUFetchError)
			},
			expectedLogs: []testlog.ExpectedLogEntry{},
			wantErrs:     []error{computeskus.ErrListVMResourceSKUs, errTestSKUFetchError},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			loadBalancers := mock_armnetwork.NewMockLoadBalancersClient(ctrl)
			skus := mock_armcompute.NewMockResourceSKUsClient(ctrl)
			plses := mock_armnetwork.NewMockPrivateLinkServicesClient(ctrl)

			env := mock_env.NewMockInterface(ctrl)
			env.EXPECT().FeatureIsSet(gomock.Any()).AnyTimes().Return(false)
			env.EXPECT().Now().AnyTimes().Return(time.Unix(1756868836, 0))

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
						APIServerProfile: api.APIServerProfile{
							IntIP: "127.1.2.3",
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

			tt.mocks(loadBalancers, skus, plses)

			manager := manager{
				doc:                           doc,
				db:                            openShiftClustersDatabase,
				log:                           entry,
				armLoadBalancers:              loadBalancers,
				armClusterPrivateLinkServices: plses,
				armResourceSKUs:               skus,
				env:                           env,
			}

			err = manager.migrateInternalLoadBalancerZones(ctx)
			utilerror.AssertErrorMatchesAll(t, err, tt.wantErrs)

			err = testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestUpdateLoadBalancerZonalMigration(t *testing.T) {
	ctx := context.Background()
	infraID := "infraID"
	location := "eastus"
	rgName := "clusterRG"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/" + rgName
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	for _, tt := range []struct {
		name                string
		architectureVersion api.ArchitectureVersion
		internalLBName      string
		backendPoolName     string
		mocks               func(lbs *mock_armnetwork.MockLoadBalancersClient, sku *mock_armcompute.MockResourceSKUsClient, plses *mock_armnetwork.MockPrivateLinkServicesClient)
		wantErr             error
		expectedLogs        []testlog.ExpectedLogEntry
	}{
		{
			name:                "performed, zonal, v2",
			architectureVersion: api.ArchitectureVersionV2,
			internalLBName:      infraID + "-internal",
			backendPoolName:     infraID,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("load balancer zonal migration: starting critical section"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("associating temporary frontend IP (1756868836-ip) to PLS"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("removing old frontend IP"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("updating internal load balancer with zone-redundant frontend IP"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("reassociating frontend IP with PLS"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("cleaning up temporary frontend IP"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("critical section complete, api-int migrated"),
				},
			},
		},
		{
			name:                "performed, zonal, v1",
			architectureVersion: api.ArchitectureVersionV1,
			internalLBName:      infraID + "-internal-lb",
			backendPoolName:     infraID + "-internal-controlplane-v4",
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("load balancer zonal migration: starting critical section"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("associating temporary frontend IP (1756868836-ip) to PLS"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("removing old frontend IP"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("updating internal load balancer with zone-redundant frontend IP"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("reassociating frontend IP with PLS"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("cleaning up temporary frontend IP"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("critical section complete, api-int migrated"),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			lbs := mock_armnetwork.NewMockLoadBalancersClient(ctrl)
			skus := mock_armcompute.NewMockResourceSKUsClient(ctrl)
			plses := mock_armnetwork.NewMockPrivateLinkServicesClient(ctrl)

			env := mock_env.NewMockInterface(ctrl)
			env.EXPECT().FeatureIsSet(gomock.Any()).AnyTimes().Return(false)
			env.EXPECT().Now().AnyTimes().Return(time.Unix(1756868836, 0))

			lbs.EXPECT().Get(gomock.Any(), rgName, tt.internalLBName, nil).Return(
				armnetwork.LoadBalancersClientGetResponse{
					LoadBalancer: armnetwork.LoadBalancer{
						ID:   new(tt.internalLBName),
						Name: new(tt.internalLBName),
						Properties: &armnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
								{
									Name:  new("internal-lb-ip-v4"),
									Zones: []*string{},
								},
							},
							LoadBalancingRules: []*armnetwork.LoadBalancingRule{
								{
									ID: new("rule1"),
								},
							},
						},
					},
				}, nil,
			)

			plses.EXPECT().Get(gomock.Any(), rgName, infraID+"-pls", nil).Return(
				armnetwork.PrivateLinkServicesClientGetResponse{
					PrivateLinkService: armnetwork.PrivateLinkService{
						Properties: &armnetwork.PrivateLinkServiceProperties{
							LoadBalancerFrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
								{
									ID: new("oldfrontendIP"),
								},
							},
						},
					},
				}, nil,
			)

			skus.EXPECT().List(gomock.Any(), "location eq eastus", false).Return([]*armcompute.ResourceSKU{
				{
					Name:      pointerutils.ToPtr(string(api.VMSizeStandardD16asV4)),
					Locations: pointerutils.ToSlicePtr([]string{"eastus"}),
					LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
						{Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"})},
					}),
					Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
					ResourceType: new("virtualMachines"),
				},
			}, nil)

			plsFIPRemoval := plses.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, infraID+"-pls", armnetwork.PrivateLinkService{
				Properties: &armnetwork.PrivateLinkServiceProperties{
					LoadBalancerFrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
						{
							ID: new(tt.internalLBName + "/frontendIPConfigurations/1756868836-ip"),
						},
					},
				},
			}, nil).Return(nil)

			bogusFIPCreation := lbs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, tt.internalLBName,
				armnetwork.LoadBalancer{
					Name: new(tt.internalLBName),
					ID:   new(tt.internalLBName),
					Properties: &armnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
							{
								Name:  new("internal-lb-ip-v4"),
								Zones: []*string{},
							},
							{
								Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
									PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
									Subnet: &armnetwork.Subnet{
										ID: new("subnetID"),
									},
								},
								Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
								Name:  new("1756868836-ip"),
							},
						},
						// when the bogus FIP is created, existing rules are maintained
						LoadBalancingRules: []*armnetwork.LoadBalancingRule{
							{
								ID: new("rule1"),
							},
						},
					},
				}, nil).Return(nil)

			ruleDeletion := lbs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, tt.internalLBName,
				armnetwork.LoadBalancer{
					Name: new(tt.internalLBName),
					ID:   new(tt.internalLBName),
					Properties: &armnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
							{
								Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
									PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
									Subnet: &armnetwork.Subnet{
										ID: new("subnetID"),
									},
								},
								Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
								Name:  new("1756868836-ip"),
							},
						},
						LoadBalancingRules: []*armnetwork.LoadBalancingRule{},
					},
				}, nil).Return(nil).After(bogusFIPCreation)

			goodFIP := &armnetwork.FrontendIPConfiguration{
				Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
					PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodStatic),
					Subnet: &armnetwork.Subnet{
						ID: new("subnetID"),
					},
					PrivateIPAddress: new("127.1.2.3"),
				},
				Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
				Name:  new(internalLBFrontendIPName),
			}

			goodRules := []*armnetwork.LoadBalancingRule{
				{
					Name: new("api-internal-v4"),
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{
							ID: new(tt.internalLBName + "/frontendIPConfigurations/internal-lb-ip-v4"),
						},
						BackendAddressPool: &armnetwork.SubResource{
							ID: new(tt.internalLBName + "/backendAddressPools/" + tt.backendPoolName),
						},
						Probe: &armnetwork.SubResource{
							ID: new(tt.internalLBName + "/probes/api-internal-probe"),
						},
						Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
						LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
						FrontendPort:         new(int32(6443)),
						BackendPort:          new(int32(6443)),
						IdleTimeoutInMinutes: new(int32(30)),
						DisableOutboundSnat:  new(true),
					},
				},
				{
					Name: new("sint-v4"),
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{
							ID: new(tt.internalLBName + "/frontendIPConfigurations/internal-lb-ip-v4"),
						},
						BackendAddressPool: &armnetwork.SubResource{
							ID: new(tt.internalLBName + "/backendAddressPools/" + tt.backendPoolName),
						},
						Probe: &armnetwork.SubResource{
							ID: new(tt.internalLBName + "/probes/sint-probe"),
						},
						Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
						LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
						FrontendPort:         new(int32(22623)),
						BackendPort:          new(int32(22623)),
						IdleTimeoutInMinutes: new(int32(30)),
					},
				},
			}

			newRulesCreation := lbs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, tt.internalLBName, armnetwork.LoadBalancer{
				ID:   new(tt.internalLBName),
				Name: new(tt.internalLBName),
				Properties: &armnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
						{
							Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
								PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
								Subnet: &armnetwork.Subnet{
									ID: new("subnetID"),
								},
							},
							Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"}),
							Name:  new("1756868836-ip"),
						},
						goodFIP,
					},
					LoadBalancingRules: goodRules,
				},
			}, nil).Return(nil).After(ruleDeletion)

			plses.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, infraID+"-pls", armnetwork.PrivateLinkService{
				Properties: &armnetwork.PrivateLinkServiceProperties{
					LoadBalancerFrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
						{
							ID: new(tt.internalLBName + "/frontendIPConfigurations/internal-lb-ip-v4"),
						},
					},
				},
			}, nil).Return(nil).After(plsFIPRemoval)

			lbs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, tt.internalLBName, armnetwork.LoadBalancer{
				ID:   new(tt.internalLBName),
				Name: new(tt.internalLBName),
				Properties: &armnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
						goodFIP,
					},
					LoadBalancingRules: goodRules,
				},
			}, nil).Return(nil).After(newRulesCreation)

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
						APIServerProfile: api.APIServerProfile{
							IntIP: "127.1.2.3",
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

			manager := manager{
				doc:                           doc,
				db:                            openShiftClustersDatabase,
				log:                           entry,
				armLoadBalancers:              lbs,
				armClusterPrivateLinkServices: plses,
				armResourceSKUs:               skus,
				env:                           env,
			}

			err = manager.migrateInternalLoadBalancerZones(ctx)
			utilerror.AssertErrorIs(t, err, tt.wantErr)

			err = testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
