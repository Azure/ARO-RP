package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	azsecretsdk "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestConfigureAPIServerCertificates(t *testing.T) {
	ctx := context.Background()
	clusterUUID := "512a50c8-2a43-4c2a-8fd9-a5539475df2a"

	for _, tt := range []struct {
		name              string
		clusterproperties api.OpenShiftClusterProperties
		objects           []runtime.Object
		check             func(clienthelper.Interface, Gomega) error
		wantMsg           string
		wantErr           string
	}{
		{
			name: "not found",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			objects: []runtime.Object{},
			wantErr: `TerminalError: apiservers.config.openshift.io "cluster" not found`,
		},
		{
			name: "not managed",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something.unmanaged",
				},
			},
			objects: []runtime.Object{},
			wantMsg: "apiserver certificate is not managed",
		},
		{
			name: "invalid domain",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something.",
				},
			},
			objects: []runtime.Object{},
			wantErr: `TerminalError: invalid domain "something."`,
		},
		{
			name: "secrets referenced",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			objects: []runtime.Object{
				&configv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: configv1.APIServerSpec{},
				},
			},
			check: func(i clienthelper.Interface, g Gomega) error {
				apiserver := &configv1.APIServer{}
				err := i.GetOne(ctx, types.NamespacedName{Name: "cluster"}, apiserver)
				if err != nil {
					return err
				}

				g.Expect(apiserver.Spec.ServingCerts.NamedCertificates).To(Equal([]configv1.APIServerNamedServingCert{
					{
						Names: []string{"api.something.example.com"},
						ServingCertificate: configv1.SecretNameReference{
							Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-apiserver",
						},
					},
				}))

				return nil
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Domain().AnyTimes().Return("example.com")

			_, log := testlog.New()

			builder := fake.NewClientBuilder().WithRuntimeObjects(tt.objects...)
			ch := clienthelper.NewWithClient(log, builder.Build())
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch),
				testtasks.WithOpenShiftClusterProperties(clusterUUID, tt.clusterproperties),
			)

			err := EnsureAPIServerServingCertificateConfiguration(tc)
			if tt.wantErr != "" && err != nil {
				g.Expect(err).To(MatchError(tt.wantErr))
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if tt.check != nil {
				g.Expect(tt.check(ch, g)).ToNot(HaveOccurred())
			}

			if tt.wantMsg != "" {
				g.Expect(tc.GetResultMessage()).To(Equal(tt.wantMsg))
			} else {
				g.Expect(tc.GetResultMessage()).To(BeEmpty())
			}
		})
	}
}

func TestRotateAPIServerCertificate(t *testing.T) {
	ctx := context.Background()
	clusterUUID := "512a50c8-2a43-4c2a-8fd9-a5539475df2a"
	secretName := clusterUUID + "-apiserver"

	for _, tt := range []struct {
		name              string
		clusterproperties api.OpenShiftClusterProperties
		objects           []runtime.Object
		check             func(clienthelper.Interface, Gomega) error
		secretDNSNames    []string
		secretFetches     int
		wantMsg           string
		wantErr           string
	}{
		{
			name: "not managed",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something.unmanaged",
				},
			},
			objects: []runtime.Object{},
			wantMsg: "apiserver certificate is not managed",
		},
		{
			name: "managed certificate rotated",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			objects: []runtime.Object{},
			check: func(ch clienthelper.Interface, g Gomega) error {
				for _, namespace := range []string{"openshift-config", "openshift-azure-operator"} {
					secret := &corev1.Secret{}
					err := ch.GetOne(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, secret)
					if err != nil {
						return err
					}

					g.Expect(secret.Type).To(Equal(corev1.SecretTypeTLS))
					g.Expect(secret.Data[corev1.TLSCertKey]).ToNot(BeEmpty())
					g.Expect(secret.Data[corev1.TLSPrivateKeyKey]).ToNot(BeEmpty())
				}
				return nil
			},
			secretDNSNames: []string{"api.something.example.com"},
			secretFetches:  3,
		},
		{
			name: "managed certificate rotated with duplicate SAN entries",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			objects: []runtime.Object{},
			check: func(ch clienthelper.Interface, g Gomega) error {
				for _, namespace := range []string{"openshift-config", "openshift-azure-operator"} {
					secret := &corev1.Secret{}
					err := ch.GetOne(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, secret)
					if err != nil {
						return err
					}

					g.Expect(secret.Type).To(Equal(corev1.SecretTypeTLS))
					g.Expect(secret.Data[corev1.TLSCertKey]).ToNot(BeEmpty())
					g.Expect(secret.Data[corev1.TLSPrivateKeyKey]).ToNot(BeEmpty())
				}
				return nil
			},
			secretDNSNames: []string{"api.something.example.com", "api.something.example.com"},
			secretFetches:  3,
		},
		{
			name: "custom certificate - skip rotation",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			objects:        []runtime.Object{},
			secretDNSNames: []string{"custom.example.com"},
			secretFetches:  1,
			wantMsg:        "apiserver certificate is custom; skipping rotation",
		},
		{
			name: "custom certificate - skip rotation when extra SAN present",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			objects:        []runtime.Object{},
			secretDNSNames: []string{"api.something.example.com", "custom.example.com"},
			secretFetches:  1,
			wantMsg:        "apiserver certificate is custom; skipping rotation",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Domain().AnyTimes().Return("example.com")

			if len(tt.secretDNSNames) > 0 {
				if tt.secretFetches == 0 {
					t.Fatalf("secretFetches must be set when secretDNSNames are provided")
				}

				kv := mock_azsecrets.NewMockClient(controller)
				_env.EXPECT().ClusterKeyvault().Return(kv)

				secretValue := newCertificatePEM(t, tt.secretDNSNames)
				kv.EXPECT().GetSecret(gomock.Any(), secretName, "", gomock.Nil()).
					Return(azsecretsdk.GetSecretResponse{
						Secret: azsecretsdk.Secret{
							Value: &secretValue,
						},
					}, nil).Times(tt.secretFetches)
			}

			_, log := testlog.New()

			builder := fake.NewClientBuilder().WithRuntimeObjects(tt.objects...)
			ch := clienthelper.NewWithClient(log, builder.Build())
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch),
				testtasks.WithOpenShiftClusterProperties(clusterUUID, tt.clusterproperties),
			)

			err := RotateAPIServerCertificate(tc)
			if tt.wantErr != "" && err != nil {
				g.Expect(err).To(MatchError(tt.wantErr))
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if tt.check != nil {
				g.Expect(tt.check(ch, g)).ToNot(HaveOccurred())
			}

			if tt.wantMsg != "" {
				g.Expect(tc.GetResultMessage()).To(Equal(tt.wantMsg))
			} else {
				g.Expect(tc.GetResultMessage()).To(BeEmpty())
			}
		})
	}
}

func newCertificatePEM(t *testing.T, dnsNames []string) string {
	t.Helper()

	if len(dnsNames) == 0 {
		t.Fatal("dnsNames must not be empty")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		DNSNames:              dnsNames,
		Subject:               pkix.Name{CommonName: dnsNames[0]},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if keyPEM == nil {
		t.Fatal("failed to encode private key")
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
	if certPEM == nil {
		t.Fatal("failed to encode certificate")
	}

	return string(keyPEM) + string(certPEM)
}
