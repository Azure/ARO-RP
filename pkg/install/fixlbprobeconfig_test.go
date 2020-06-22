package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/tls"
)

func TestFixLBProbes(t *testing.T) {
	ctx := context.Background()
	subscriptionID := "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"

	_, goodCert, err := tls.GenerateTestKeyAndCertificate("good", nil, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, malformedCert, err := tls.GenerateTestKeyAndCertificate("malformed", nil, nil, false, false, func(cert *x509.Certificate) {
		cert.SubjectKeyId = []byte{1}
		cert.AuthorityKeyId = cert.SubjectKeyId
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		infraID string
		cert    *x509.Certificate
		mocks   func(*mock_network.MockLoadBalancersClient)
		wantErr string
	}{
		{
			name:    "private/good",
			infraID: "test",
			cert:    goodCert[0],
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

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "test-public-lb", "").Return(
					mgmtnetwork.LoadBalancer{}, nil)

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
			name:    "private/malformed",
			infraID: "test",
			cert:    malformedCert[0],
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
					}, nil)

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "test-public-lb", "").Return(
					mgmtnetwork.LoadBalancer{}, nil)

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
			},
		},
		{
			name: "public/good",
			cert: goodCert[0],
			mocks: func(lbc *mock_network.MockLoadBalancersClient) {
				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "aro-internal-lb", "").Return(
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

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "aro-internal-lb",
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

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "aro-public-lb", "").Return(
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

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "aro-public-lb",
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
		{
			name: "public/malformed",
			cert: malformedCert[0],
			mocks: func(lbc *mock_network.MockLoadBalancersClient) {
				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "aro-internal-lb", "").Return(
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
					}, nil)

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "aro-internal-lb",
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

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "aro-public-lb", "").Return(
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

				lbc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "aro-public-lb",
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
		{
			name: "public/good no change",
			cert: goodCert[0],
			mocks: func(lbc *mock_network.MockLoadBalancersClient) {
				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "aro-internal-lb", "").Return(
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
					}, nil)

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "aro-public-lb", "").Return(
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
					}, nil)
			},
		},
		{
			name: "private/malformed no change",
			cert: malformedCert[0],
			mocks: func(lbc *mock_network.MockLoadBalancersClient) {
				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "aro-internal-lb", "").Return(
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

				lbc.EXPECT().Get(gomock.Any(), "test-cluster", "aro-public-lb", "").Return(
					mgmtnetwork.LoadBalancer{}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			kubernetescli := fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "openshift-machine-config-operator",
					Name:      "machine-config-server-tls",
				},
				Data: map[string][]byte{
					v1.TLSCertKey: pem.EncodeToMemory(&pem.Block{
						Type:  "CERTIFICATE",
						Bytes: tt.cert.Raw,
					}),
				},
			})

			loadbalancersClient := mock_network.NewMockLoadBalancersClient(controller)
			tt.mocks(loadbalancersClient)

			i := &Installer{
				kubernetescli: kubernetescli,
				loadbalancers: loadbalancersClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							InfraID: tt.infraID,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", subscriptionID),
							},
						},
					},
				},
			}

			err := i.fixLBProbes(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
