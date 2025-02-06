package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net"
	"testing"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/util/installer"
	mock_graph "github.com/Azure/ARO-RP/pkg/util/mocks/graph"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

func TestFixMCSCert(t *testing.T) {
	ctx := context.Background()

	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(validCaCerts[0])

	for _, tt := range []struct {
		name             string
		manager          func(*gomock.Controller, *bool) (*manager, error)
		wantDeleteCalled bool
	}{
		{
			name: "basic",
			manager: func(controller *gomock.Controller, deleteCalled *bool) (*manager, error) {
				b := x509.MarshalPKCS1PrivateKey(validCaKey)

				_, validCerts, err := utiltls.GenerateKeyAndCertificate("cert", validCaKey, validCaCerts[0], false, false)
				if err != nil {
					t.Fatal(err)
				}

				pg := graph.PersistedGraph{}
				root := &installer.RootCA{
					SelfSignedCertKey: installer.SelfSignedCertKey{
						CertKey: installer.CertKey{
							CertRaw: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validCaCerts[0].Raw}),
							KeyRaw:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
						},
					},
				}

				data, err := json.Marshal(root)
				if err != nil {
					return nil, err
				}

				pg["*tls.RootCA"] = data
				graph := mock_graph.NewMockManager(controller)
				graph.EXPECT().LoadPersisted(ctx, "", "cluster").Return(pg, nil)

				kubernetescli := fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "machine-config-server-tls",
						Namespace: "openshift-machine-config-operator",
					},
					Data: map[string][]byte{
						corev1.TLSCertKey: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validCerts[0].Raw}),
					},
				})
				kubernetescli.AddReactor("delete-collection", "pods", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					if action, ok := action.(ktesting.DeleteCollectionAction); ok {
						if action.GetListRestrictions().Labels.String() == "k8s-app=machine-config-server" {
							*deleteCalled = true
						}
					}
					return false, nil, nil
				})

				return &manager{
					doc: &api.OpenShiftClusterDocument{
						OpenShiftCluster: &api.OpenShiftCluster{
							Properties: api.OpenShiftClusterProperties{
								ClusterProfile: api.ClusterProfile{
									Domain: "foo.bar",
								},
								APIServerProfile: api.APIServerProfile{
									IntIP: "10.0.0.1",
								},
							},
						},
					},
					graph:         graph,
					kubernetescli: kubernetescli,
				}, nil
			},
			wantDeleteCalled: true,
		},
		{
			name: "noop",
			manager: func(controller *gomock.Controller, deleteCalled *bool) (*manager, error) {
				validKey, validCerts, err := utiltls.GenerateTestKeyAndCertificate("system:machine-config-server", validCaKey, validCaCerts[0], false, false, func(template *x509.Certificate) {
					template.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
				})
				if err != nil {
					t.Fatal(err)
				}

				b := x509.MarshalPKCS1PrivateKey(validKey)

				return &manager{
					doc: &api.OpenShiftClusterDocument{
						OpenShiftCluster: &api.OpenShiftCluster{
							Properties: api.OpenShiftClusterProperties{
								ClusterProfile: api.ClusterProfile{
									Domain: "foo.bar",
								},
								APIServerProfile: api.APIServerProfile{
									IntIP: "10.0.0.1",
								},
							},
						},
					},
					kubernetescli: fake.NewSimpleClientset(&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "machine-config-server-tls",
							Namespace: "openshift-machine-config-operator",
						},
						Data: map[string][]byte{
							corev1.TLSCertKey:       pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validCerts[0].Raw}),
							corev1.TLSPrivateKeyKey: pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}),
						},
					}),
				}, nil
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			var deleteCalled bool
			m, err := tt.manager(controller, &deleteCalled)
			if err != nil {
				t.Error(err)
			}

			err = m.fixMCSCert(ctx)
			if err != nil {
				t.Error(err)
			}

			if deleteCalled != tt.wantDeleteCalled {
				t.Error(deleteCalled)
			}

			s, err := m.kubernetescli.CoreV1().Secrets("openshift-machine-config-operator").Get(ctx, "machine-config-server-tls", metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			var pemdata []byte
			pemdata = append(pemdata, s.Data[corev1.TLSCertKey]...)
			pemdata = append(pemdata, s.Data[corev1.TLSPrivateKeyKey]...)

			key, certs, err := utilpem.Parse(pemdata)
			if err != nil {
				t.Error(err)
			}

			cert := certs[0]

			_, err = cert.Verify(x509.VerifyOptions{
				Roots: pool,
			})
			if err != nil {
				t.Error(err)
			}

			if !cert.PublicKey.(*rsa.PublicKey).Equal(&key.PublicKey) {
				t.Error("key mismatch")
			}

			if cert.Subject.String() != "CN=system:machine-config-server" {
				t.Error(cert.Subject)
			}

			if !cert.IPAddresses[0].Equal(net.ParseIP("10.0.0.1")) {
				t.Error(cert.IPAddresses[0])
			}
		})
	}
}
