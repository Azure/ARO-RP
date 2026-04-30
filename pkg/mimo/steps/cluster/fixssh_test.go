package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	armnetwork_sdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func genInt(name string, withPool []*armnetwork_sdk.BackendAddressPool) *armnetwork_sdk.Interface {
	i := &armnetwork_sdk.Interface{
		Name: pointerutils.ToPtr(name),
		Properties: &armnetwork_sdk.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork_sdk.InterfaceIPConfiguration{
				{
					Name: pointerutils.ToPtr(name),
					Properties: &armnetwork_sdk.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &armnetwork_sdk.Subnet{Name: pointerutils.ToPtr("subnetID")},
					},
				},
			},
			VirtualMachine: &armnetwork_sdk.SubResource{
				ID: pointerutils.ToPtr("somemachine"),
			},
		},
	}

	if len(withPool) > 0 {
		i.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools = withPool
	}
	return i
}

// TestFixSSH ensures that the usual FixSSH works inside MIMO. See
// pkg/cluster/fixssh_test.go for more indepth testing.
func TestFixSSH(t *testing.T) {
	ctx := t.Context()
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
			name:                "normal, v2",
			architectureVersion: api.ArchitectureVersionV2,
			internalLBName:      infraID + "-internal",
			backendPoolName:     infraID,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding SSH Backend Address Pool ssh-0 to Internal Load Balancer infraID-internal"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding SSH Backend Address Pool ssh-1 to Internal Load Balancer infraID-internal"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding SSH Backend Address Pool ssh-2 to Internal Load Balancer infraID-internal"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding SSH Load Balancing Rule for ssh-0 to Internal Load Balancer infraID-internal"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding SSH Load Balancing Rule for ssh-1 to Internal Load Balancer infraID-internal"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding SSH Load Balancing Rule for ssh-2 to Internal Load Balancer infraID-internal"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding ssh Health Probe to Internal Load Balancer infraID-internal"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Updating Load Balancer infraID-internal"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Checking NIC infraID-master0-nic"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding NIC infraID-master0-nic to Internal Load Balancer API Address Pool infraID-internal/backendAddressPools/infraID"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding NIC infraID-master0-nic to Internal Load Balancer SSH Address Pool infraID-internal/backendAddressPools/ssh-0"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("not updating external LB address pool assignment as this is a UDR cluster"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Updating Network Interface infraID-master0-nic"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Checking NIC infraID-master1-nic"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding NIC infraID-master1-nic to Internal Load Balancer API Address Pool infraID-internal/backendAddressPools/infraID"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding NIC infraID-master1-nic to Internal Load Balancer SSH Address Pool infraID-internal/backendAddressPools/ssh-1"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("not updating external LB address pool assignment as this is a UDR cluster"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Updating Network Interface infraID-master1-nic"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Checking NIC infraID-master2-nic"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding NIC infraID-master2-nic to Internal Load Balancer API Address Pool infraID-internal/backendAddressPools/infraID"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Adding NIC infraID-master2-nic to Internal Load Balancer SSH Address Pool infraID-internal/backendAddressPools/ssh-2"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("not updating external LB address pool assignment as this is a UDR cluster"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Updating Network Interface infraID-master2-nic"),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			lbs := mock_armnetwork.NewMockLoadBalancersClient(ctrl)
			skus := mock_armcompute.NewMockResourceSKUsClient(ctrl)
			ints := mock_armnetwork.NewMockInterfacesClient(ctrl)

			env := mock_env.NewMockInterface(ctrl)
			env.EXPECT().FeatureIsSet(gomock.Any()).AnyTimes().Return(false)
			env.EXPECT().Now().AnyTimes().Return(time.Unix(1756868836, 0))

			ints.EXPECT().List(gomock.Any(), rgName, gomock.Any()).Return(
				[]*armnetwork_sdk.Interface{
					genInt(infraID+"-master0-nic", nil), genInt(infraID+"-master1-nic", nil), genInt(infraID+"-master2-nic", nil),
				}, nil)

			ints.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, infraID+"-master0-nic", *genInt(infraID+"-master0-nic", []*armnetwork_sdk.BackendAddressPool{
				{ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/infraID")},
				{ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/ssh-0")},
			}), gomock.Any()).Return(nil)

			ints.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, infraID+"-master1-nic", *genInt(infraID+"-master1-nic", []*armnetwork_sdk.BackendAddressPool{
				{ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/infraID")},
				{ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/ssh-1")},
			}), gomock.Any()).Return(nil)
			ints.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, infraID+"-master2-nic", *genInt(infraID+"-master2-nic", []*armnetwork_sdk.BackendAddressPool{
				{ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/infraID")},
				{ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/ssh-2")},
			}), gomock.Any()).Return(nil)

			lbs.EXPECT().Get(gomock.Any(), rgName, tt.internalLBName, nil).Return(
				armnetwork_sdk.LoadBalancersClientGetResponse{
					LoadBalancer: armnetwork_sdk.LoadBalancer{
						ID:   pointerutils.ToPtr(tt.internalLBName),
						Name: pointerutils.ToPtr(tt.internalLBName),
						Properties: &armnetwork_sdk.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: []*armnetwork_sdk.FrontendIPConfiguration{
								{
									ID:    pointerutils.ToPtr(tt.internalLBName + "/frontendIPConfigurations/internal-lb-ip-v4"),
									Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
									Zones: []*string{},
								},
							},
							LoadBalancingRules: []*armnetwork_sdk.LoadBalancingRule{},
						},
					},
				}, nil,
			)

			goodRules := []*armnetwork_sdk.LoadBalancingRule{
				{
					Name: pointerutils.ToPtr("ssh-0"),
					Properties: &armnetwork_sdk.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/frontendIPConfigurations/internal-lb-ip-v4"),
						},
						BackendAddressPool: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/ssh-0"),
						},
						Probe: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/probes/ssh"),
						},
						Protocol:             pointerutils.ToPtr(armnetwork_sdk.TransportProtocolTCP),
						LoadDistribution:     pointerutils.ToPtr(armnetwork_sdk.LoadDistributionDefault),
						FrontendPort:         pointerutils.ToPtr(int32(2200)),
						BackendPort:          pointerutils.ToPtr(int32(22)),
						IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
						DisableOutboundSnat:  pointerutils.ToPtr(true),
					},
				},

				{
					Name: pointerutils.ToPtr("ssh-1"),
					Properties: &armnetwork_sdk.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/frontendIPConfigurations/internal-lb-ip-v4"),
						},
						BackendAddressPool: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/ssh-1"),
						},
						Probe: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/probes/ssh"),
						},
						Protocol:             pointerutils.ToPtr(armnetwork_sdk.TransportProtocolTCP),
						LoadDistribution:     pointerutils.ToPtr(armnetwork_sdk.LoadDistributionDefault),
						FrontendPort:         pointerutils.ToPtr(int32(2201)),
						BackendPort:          pointerutils.ToPtr(int32(22)),
						IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
						DisableOutboundSnat:  pointerutils.ToPtr(true),
					},
				},
				{
					Name: pointerutils.ToPtr("ssh-2"),
					Properties: &armnetwork_sdk.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/frontendIPConfigurations/internal-lb-ip-v4"),
						},
						BackendAddressPool: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/backendAddressPools/ssh-2"),
						},
						Probe: &armnetwork_sdk.SubResource{
							ID: pointerutils.ToPtr(tt.internalLBName + "/probes/ssh"),
						},
						Protocol:             pointerutils.ToPtr(armnetwork_sdk.TransportProtocolTCP),
						LoadDistribution:     pointerutils.ToPtr(armnetwork_sdk.LoadDistributionDefault),
						FrontendPort:         pointerutils.ToPtr(int32(2202)),
						BackendPort:          pointerutils.ToPtr(int32(22)),
						IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
						DisableOutboundSnat:  pointerutils.ToPtr(true),
					},
				},
			}
			lbs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), rgName, tt.internalLBName, armnetwork_sdk.LoadBalancer{
				ID:   pointerutils.ToPtr(tt.internalLBName),
				Name: pointerutils.ToPtr(tt.internalLBName),
				Properties: &armnetwork_sdk.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: []*armnetwork_sdk.FrontendIPConfiguration{
						{
							ID:    pointerutils.ToPtr(tt.internalLBName + "/frontendIPConfigurations/internal-lb-ip-v4"),
							Zones: []*string{},
							Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
						},
					},
					BackendAddressPools: []*armnetwork_sdk.BackendAddressPool{
						{
							Name: pointerutils.ToPtr("ssh-0"),
						},
						{
							Name: pointerutils.ToPtr("ssh-1"),
						},
						{
							Name: pointerutils.ToPtr("ssh-2"),
						},
					},
					LoadBalancingRules: goodRules,
					Probes: []*armnetwork_sdk.Probe{
						{
							Name: pointerutils.ToPtr("ssh"),
							Properties: &armnetwork_sdk.ProbePropertiesFormat{
								Protocol:          pointerutils.ToPtr(armnetwork_sdk.ProbeProtocolTCP),
								Port:              pointerutils.ToPtr(int32(22)),
								IntervalInSeconds: pointerutils.ToPtr(int32(5)),
								NumberOfProbes:    pointerutils.ToPtr(int32(2)),
							},
						},
					},
				},
			}, gomock.Any()).Return(nil)

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
							OutboundType:        api.OutboundTypeUserDefinedRouting,
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
			g := gomega.NewWithT(t)
			hook, entry := testlog.New()

			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(doc)
			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			tc := testtasks.NewFakeTestContext(
				ctx, env, entry, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithOpenShiftDatabase(openShiftClustersDatabase),
				testtasks.WithOpenShiftClusterDocument(doc),
				testtasks.WithLoadBalancersClient(lbs),
				testtasks.WithResourceSKUsClient(skus),
				testtasks.WithInterfacesClient(ints),
			)

			err = FixSSHStep(tc)

			if tt.wantErr != nil && err != nil {
				g.Expect(err).To(gomega.MatchError(tt.wantErr))
			} else if tt.wantErr != nil && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == nil {
				g.Expect(err).ToNot(gomega.HaveOccurred())
			}

			err = testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
