package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateLoadBalancerProfile(t *testing.T) {
	location := "eastus"
	clusterRGID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/clusterRG"
	clusterRGName := "clusterRG"
	infraID := "infraID"

	for _, tt := range []struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(spNetworkUsage *mock_network.MockUsageClient, loadBalancerBackendAddressPoolsClient *mock_network.MockLoadBalancerBackendAddressPoolsClient)
		wantErr string
	}{
		{
			name: "validation skipped outboundType is UserDefinedRouting",
			oc: &api.OpenShiftCluster{
				Location: location,
				Properties: api.OpenShiftClusterProperties{
					NetworkProfile: api.NetworkProfile{
						OutboundType: api.OutboundTypeUserDefinedRouting,
					},
				},
			},
		},
		{
			name: "validation executed",
			oc: &api.OpenShiftCluster{
				Location: location,
				Properties: api.OpenShiftClusterProperties{
					InfraID: infraID,
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: clusterRGID,
					},
					ProvisioningState: api.ProvisioningStateUpdating,
					NetworkProfile: api.NetworkProfile{
						LoadBalancerProfile: &api.LoadBalancerProfile{
							EffectiveOutboundIPs: []api.EffectiveOutboundIP{
								{
									ID: "managed-ip-1",
								},
								{
									ID: "managed-ip-2",
								},
								{
									ID: "managed-ip-3",
								},
							},
							ManagedOutboundIPs: &api.ManagedOutboundIPs{
								Count: 3,
							},
						},
					},
				},
			},
			mocks: func(spNetworkUsage *mock_network.MockUsageClient,
				loadBalancerBackendAddressPoolsClient *mock_network.MockLoadBalancerBackendAddressPoolsClient) {
				spNetworkUsage.EXPECT().
					List(gomock.Any(), location).
					Return([]mgmtnetwork.Usage{
						{
							Name: &mgmtnetwork.UsageName{
								Value: to.StringPtr("PublicIPAddresses"),
							},
							CurrentValue: to.Int64Ptr(4),
							Limit:        to.Int64Ptr(10),
						},
					}, nil)
				loadBalancerBackendAddressPoolsClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, infraID).
					Return(mgmtnetwork.BackendAddressPool{
						BackendAddressPoolPropertiesFormat: &mgmtnetwork.BackendAddressPoolPropertiesFormat{
							BackendIPConfigurations: getFakeBackendIPConfigs(6),
						},
					}, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			controller := gomock.NewController(t)
			defer controller.Finish()

			loadBalancerBackendAddressPoolsClient := mock_network.NewMockLoadBalancerBackendAddressPoolsClient(controller)
			networkUsageClient := mock_network.NewMockUsageClient(controller)

			if tt.mocks != nil {
				tt.mocks(networkUsageClient, loadBalancerBackendAddressPoolsClient)
			}
			dv := &dynamic{
				log:                                   logrus.NewEntry(logrus.StandardLogger()),
				spNetworkUsage:                        networkUsageClient,
				loadBalancerBackendAddressPoolsClient: loadBalancerBackendAddressPoolsClient,
			}

			err := dv.ValidateLoadBalancerProfile(ctx, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidatePublicIPQuota(t *testing.T) {
	clusterRGID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/clusterRG"
	location := "eastus"

	for _, tt := range []struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(spNetworkUsage *mock_network.MockUsageClient)
		wantErr string
	}{
		{
			name: "cluster update - sufficient IP quota",
			oc: &api.OpenShiftCluster{
				Location: location,
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: clusterRGID,
					},
					ProvisioningState: api.ProvisioningStateUpdating,
					NetworkProfile: api.NetworkProfile{
						LoadBalancerProfile: &api.LoadBalancerProfile{
							EffectiveOutboundIPs: []api.EffectiveOutboundIP{
								{
									ID: "managed-ip-1",
								},
								{
									ID: "managed-ip-2",
								},
								{
									ID: "managed-ip-3",
								},
							},
							ManagedOutboundIPs: &api.ManagedOutboundIPs{
								Count: 5,
							},
						},
					},
				},
			},
			mocks: func(spNetworkUsage *mock_network.MockUsageClient) {
				spNetworkUsage.EXPECT().
					List(gomock.Any(), location).
					Return([]mgmtnetwork.Usage{
						{
							Name: &mgmtnetwork.UsageName{
								Value: to.StringPtr("PublicIPAddresses"),
							},
							CurrentValue: to.Int64Ptr(4),
							Limit:        to.Int64Ptr(10),
						},
					}, nil)
			},
		},
		{
			name: "cluster update - insufficient IP quota",
			oc: &api.OpenShiftCluster{
				Location: "eastus",
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: clusterRGID,
					},
					ProvisioningState: api.ProvisioningStateUpdating,
					NetworkProfile: api.NetworkProfile{
						LoadBalancerProfile: &api.LoadBalancerProfile{
							EffectiveOutboundIPs: []api.EffectiveOutboundIP{
								{
									ID: "managed-ip-1",
								},
								{
									ID: "managed-ip-2",
								},
								{
									ID: "managed-ip-3",
								},
							},
							ManagedOutboundIPs: &api.ManagedOutboundIPs{
								Count: 6,
							},
						},
					},
				},
			},
			mocks: func(spNetworkUsage *mock_network.MockUsageClient) {
				spNetworkUsage.EXPECT().
					List(gomock.Any(), location).
					Return([]mgmtnetwork.Usage{
						{
							Name: &mgmtnetwork.UsageName{
								Value: to.StringPtr("PublicIPAddresses"),
							},
							CurrentValue: to.Int64Ptr(8),
							Limit:        to.Int64Ptr(10),
						},
					}, nil)
			},
			wantErr: "400: ResourceQuotaExceeded: properties.networkProfile.loadBalancerProfile.ManagedOutboundIPs.Count: Resource quota of PublicIPAddresses exceeded. Maximum allowed: 10, Current in use: 8, Additional requested: 3.",
		},
		{
			name: "cluster create - sufficient IP quota",
			oc: &api.OpenShiftCluster{
				Location: location,
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: clusterRGID,
					},
					IngressProfiles: []api.IngressProfile{
						{
							Visibility: api.VisibilityPublic,
						},
					},
					ProvisioningState: api.ProvisioningStateCreating,
					NetworkProfile: api.NetworkProfile{
						LoadBalancerProfile: &api.LoadBalancerProfile{
							EffectiveOutboundIPs: []api.EffectiveOutboundIP{},
							ManagedOutboundIPs: &api.ManagedOutboundIPs{
								Count: 5,
							},
						},
					},
				},
			},
			mocks: func(spNetworkUsage *mock_network.MockUsageClient) {
				spNetworkUsage.EXPECT().
					List(gomock.Any(), location).
					Return([]mgmtnetwork.Usage{
						{
							Name: &mgmtnetwork.UsageName{
								Value: to.StringPtr("PublicIPAddresses"),
							},
							CurrentValue: to.Int64Ptr(4),
							Limit:        to.Int64Ptr(10),
						},
					}, nil)
			},
		},
		{
			name: "cluster create - insufficient IP quota",
			oc: &api.OpenShiftCluster{
				Location: "eastus",
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: clusterRGID,
					},
					IngressProfiles: []api.IngressProfile{
						{
							Visibility: api.VisibilityPublic,
						},
					},
					ProvisioningState: api.ProvisioningStateCreating,
					NetworkProfile: api.NetworkProfile{
						LoadBalancerProfile: &api.LoadBalancerProfile{
							EffectiveOutboundIPs: []api.EffectiveOutboundIP{},
							ManagedOutboundIPs: &api.ManagedOutboundIPs{
								Count: 6,
							},
						},
					},
				},
			},
			mocks: func(spNetworkUsage *mock_network.MockUsageClient) {
				spNetworkUsage.EXPECT().
					List(gomock.Any(), location).
					Return([]mgmtnetwork.Usage{
						{
							Name: &mgmtnetwork.UsageName{
								Value: to.StringPtr("PublicIPAddresses"),
							},
							CurrentValue: to.Int64Ptr(8),
							Limit:        to.Int64Ptr(10),
						},
					}, nil)
			},
			wantErr: "400: ResourceQuotaExceeded: properties.networkProfile.loadBalancerProfile.ManagedOutboundIPs.Count: Resource quota of PublicIPAddresses exceeded. Maximum allowed: 10, Current in use: 8, Additional requested: 7.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			controller := gomock.NewController(t)
			defer controller.Finish()

			networkUsageClient := mock_network.NewMockUsageClient(controller)

			if tt.mocks != nil {
				tt.mocks(networkUsageClient)
			}

			dv := &dynamic{
				log:            logrus.NewEntry(logrus.StandardLogger()),
				spNetworkUsage: networkUsageClient,
			}

			err := dv.validatePublicIPQuota(ctx, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateOBRuleV4FrontendPorts(t *testing.T) {
	clusterRGID := "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/clusterRG"
	clusterRGName := "clusterRG"
	location := "eastus"
	infraID := "infraID"

	for _, tt := range []struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(loadBalancerBackendAddressPoolsClient *mock_network.MockLoadBalancerBackendAddressPoolsClient)
		wantErr string
	}{
		{
			name: "valid backend pool size with IP scaling managed IPs from 2 to 1",
			oc: &api.OpenShiftCluster{
				Location: location,
				Properties: api.OpenShiftClusterProperties{
					InfraID: infraID,
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: clusterRGID,
					},
					ProvisioningState: api.ProvisioningStateUpdating,
					NetworkProfile: api.NetworkProfile{
						LoadBalancerProfile: &api.LoadBalancerProfile{
							EffectiveOutboundIPs: []api.EffectiveOutboundIP{
								{
									ID: "managed-ip-1",
								},
								{
									ID: "managed-ip-2",
								},
							},
							ManagedOutboundIPs: &api.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
				},
			},
			mocks: func(
				loadBalancerBackendAddressPoolsClient *mock_network.MockLoadBalancerBackendAddressPoolsClient) {
				loadBalancerBackendAddressPoolsClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, infraID).
					Return(mgmtnetwork.BackendAddressPool{
						BackendAddressPoolPropertiesFormat: &mgmtnetwork.BackendAddressPoolPropertiesFormat{
							BackendIPConfigurations: getFakeBackendIPConfigs(62),
						},
					}, nil)
			},
		},
		{
			name: "invalid backend pool when scaling managed IPs from 2 to 1",
			oc: &api.OpenShiftCluster{
				Location: location,
				Properties: api.OpenShiftClusterProperties{
					InfraID: infraID,
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: clusterRGID,
					},
					ProvisioningState: api.ProvisioningStateUpdating,
					NetworkProfile: api.NetworkProfile{
						LoadBalancerProfile: &api.LoadBalancerProfile{
							EffectiveOutboundIPs: []api.EffectiveOutboundIP{
								{
									ID: "managed-ip-1",
								},
								{
									ID: "managed-ip-2",
								},
							},
							ManagedOutboundIPs: &api.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
				},
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.loadBalancerProfile: Insufficient frontend ports to support the backend instance count.  Total frontend ports: 63992, Required frontend ports: 64512, Total backend instances: 63",
			mocks: func(
				loadBalancerBackendAddressPoolsClient *mock_network.MockLoadBalancerBackendAddressPoolsClient) {
				loadBalancerBackendAddressPoolsClient.EXPECT().
					Get(gomock.Any(), clusterRGName, infraID, infraID).
					Return(mgmtnetwork.BackendAddressPool{
						BackendAddressPoolPropertiesFormat: &mgmtnetwork.BackendAddressPoolPropertiesFormat{
							BackendIPConfigurations: getFakeBackendIPConfigs(63),
						},
					}, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			controller := gomock.NewController(t)
			defer controller.Finish()

			loadBalancerBackendAddressPoolsClient := mock_network.NewMockLoadBalancerBackendAddressPoolsClient(controller)

			if tt.mocks != nil {
				tt.mocks(loadBalancerBackendAddressPoolsClient)
			}

			dv := &dynamic{
				log:                                   logrus.NewEntry(logrus.StandardLogger()),
				loadBalancerBackendAddressPoolsClient: loadBalancerBackendAddressPoolsClient,
			}

			err := dv.validateOBRuleV4FrontendPorts(ctx, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func getFakeBackendIPConfigs(ipConfigCount int) *[]mgmtnetwork.InterfaceIPConfiguration {
	ipConfigs := []mgmtnetwork.InterfaceIPConfiguration{}
	for i := 0; i < ipConfigCount; i++ {
		ipConfigName := "ip-" + strconv.Itoa(i)
		ipConfigs = append(ipConfigs, mgmtnetwork.InterfaceIPConfiguration{Name: to.StringPtr(ipConfigName)})
	}
	return &ipConfigs
}
