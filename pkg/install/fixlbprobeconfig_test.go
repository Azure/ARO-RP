package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

func TestFixLBProbes(t *testing.T) {
	subscriptionID := "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"
	tests := []struct {
		name      string
		visiblity api.Visibility
		mocks     func(*mock_network.MockLoadBalancersClient)
		wantErr   bool
	}{
		{
			name:      "private",
			visiblity: api.VisibilityPrivate,
			mocks: func(lbc *mock_network.MockLoadBalancersClient) {
				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "test-internal-lb", "").Return(
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(22623),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("sint-probe"),
								},
							},
						},
					}, nil)

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "test-internal-lb",
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(22623),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("sint-probe"),
								},
							},
						},
					}).Return(nil)

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "test-internal-lb", "").Return(
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(22623),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("sint-probe"),
								},
							},
						},
					}, nil)

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "test-internal-lb",
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(22623),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/healthz"),
									},
									Name: to.StringPtr("sint-probe"),
								},
							},
						},
					}).Return(nil)

			},
		},
		{
			name:      "public",
			visiblity: api.VisibilityPublic,
			mocks: func(lbc *mock_network.MockLoadBalancersClient) {
				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "test-internal-lb", "").Return(
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(22623),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("sint-probe"),
								},
							},
						},
					}, nil)

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "test-internal-lb",
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(22623),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("sint-probe"),
								},
							},
						},
					}).Return(nil)

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "test-internal-lb", "").Return(
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(22623),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("sint-probe"),
								},
							},
						},
					}, nil)

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "test-internal-lb",
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(22623),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/healthz"),
									},
									Name: to.StringPtr("sint-probe"),
								},
							},
						},
					}).Return(nil)

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "test-public-lb", "").Return(
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolTCP,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
							},
						},
					}, nil)

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "test-public-lb",
					mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
							},
						},
					}).Return(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			ctx := context.Background()

			loadbalancersClient := mock_network.NewMockLoadBalancersClient(controller)
			i := &Installer{
				log:           logrus.NewEntry(logrus.StandardLogger()),
				loadbalancers: loadbalancersClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							InfraID:          "test",
							APIServerProfile: api.APIServerProfile{Visibility: tt.visiblity},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", subscriptionID),
							},
						},
					},
				},
			}
			tt.mocks(loadbalancersClient)
			if err := i.fixLBProbes(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Installer.fixLBProbes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
