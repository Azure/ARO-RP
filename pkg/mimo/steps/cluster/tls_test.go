package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"maps"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	azcoreruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
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
		featureDisabled   bool
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
			name: "signed certificates disabled",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			objects:         []runtime.Object{},
			featureDisabled: true,
			wantMsg:         "signed certificates disabled",
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
			_env.EXPECT().FeatureIsSet(env.FeatureDisableSignedCertificates).AnyTimes().Return(tt.featureDisabled)

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

			g.Expect(tc.GetResultMessage()).To(Equal(tt.wantMsg))

			if tt.check != nil {
				g.Expect(tt.check(ch, g)).ToNot(HaveOccurred())
			}
		})
	}
}

func TestRotateAPIServerCertificate(t *testing.T) {
	ctx := context.Background()
	clusterUUID := "512a50c8-2a43-4c2a-8fd9-a5539475df2a"

	key, certificate, err := utiltls.GenerateTestKeyAndCertificate("api-cert", nil, nil, false, false, func(c *x509.Certificate) {})
	if err != nil {
		t.Fatal(err)
	}

	ingresskey, ingresscertificate, err := utiltls.GenerateTestKeyAndCertificate("star-cert", nil, nil, false, false, func(c *x509.Certificate) {})
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name              string
		clusterproperties api.OpenShiftClusterProperties
		objects           []runtime.Object
		check             func(clienthelper.Interface, Gomega)
		mocks             func(*mock_azsecrets.MockClient)
		featureDisabled   bool
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
			wantMsg: "cluster certificates are not managed",
		},
		{
			name: "signed certificates disabled",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			featureDisabled: true,
			objects:         []runtime.Object{},
			wantMsg:         "signed certificates disabled",
		},
		{
			name: "creates certs",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			mocks: func(kv *mock_azsecrets.MockClient) {
				apiserver_secret := azsecrets.Secret{Value: pointerutils.ToPtr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate[0].Raw})) + string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})))}
				ingress_secret := azsecrets.Secret{Value: pointerutils.ToPtr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ingresscertificate[0].Raw})) + string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(ingresskey)})))}

				kv.EXPECT().GetSecret(gomock.Any(), "512a50c8-2a43-4c2a-8fd9-a5539475df2a-apiserver", "", nil).Return(
					azsecrets.GetSecretResponse{Secret: apiserver_secret}, nil,
				)
				kv.EXPECT().GetSecret(gomock.Any(), "512a50c8-2a43-4c2a-8fd9-a5539475df2a-ingress", "", nil).Return(
					azsecrets.GetSecretResponse{Secret: ingress_secret}, nil,
				)
			},
			check: func(i clienthelper.Interface, g Gomega) {
				// check API server certificate is saved correctly
				s := &corev1.Secret{}
				err := i.GetOne(ctx, types.NamespacedName{Namespace: "openshift-config", Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-apiserver"}, s)
				g.Expect(err).ShouldNot(HaveOccurred())

				s_op := &corev1.Secret{}
				err = i.GetOne(ctx, types.NamespacedName{Namespace: "openshift-azure-operator", Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-apiserver"}, s_op)
				g.Expect(err).ShouldNot(HaveOccurred())

				g.Expect(maps.Keys(s.Data)).To(ContainElements(corev1.TLSCertKey, corev1.TLSPrivateKeyKey))
				_, cert, err := utilpem.Parse(s.Data[corev1.TLSCertKey])
				g.Expect(err).ShouldNot(HaveOccurred())

				g.Expect(cert).To(HaveLen(1))
				g.Expect(cert[0].Subject.CommonName).To(Equal("api-cert"))

				// same version is in the operator and openshift-config
				g.Expect(s.Data).To(Equal(s_op.Data))

				// check ingress server certificate is saved correctly
				err = i.GetOne(ctx, types.NamespacedName{Namespace: "openshift-ingress", Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-ingress"}, s)
				g.Expect(err).ShouldNot(HaveOccurred())

				err = i.GetOne(ctx, types.NamespacedName{Namespace: "openshift-azure-operator", Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-ingress"}, s_op)
				g.Expect(err).ShouldNot(HaveOccurred())

				g.Expect(maps.Keys(s.Data)).To(ContainElements(corev1.TLSCertKey, corev1.TLSPrivateKeyKey))
				_, cert, err = utilpem.Parse(s.Data[corev1.TLSCertKey])
				g.Expect(err).ShouldNot(HaveOccurred())

				g.Expect(cert).To(HaveLen(1))
				g.Expect(cert[0].Subject.CommonName).To(Equal("star-cert"))

				// same version is in the operator and openshift-config
				g.Expect(s.Data).To(Equal(s_op.Data))
			},
			objects: []runtime.Object{},
		},
		{
			name: "missing cert still loads other",
			clusterproperties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: "something",
				},
			},
			mocks: func(kv *mock_azsecrets.MockClient) {
				ingress_secret := azsecrets.Secret{Value: pointerutils.ToPtr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ingresscertificate[0].Raw})) + string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(ingresskey)})))}

				kv.EXPECT().GetSecret(gomock.Any(), "512a50c8-2a43-4c2a-8fd9-a5539475df2a-apiserver", "", nil).Times(1).Return(
					azsecrets.GetSecretResponse{}, azcoreruntime.NewResponseError(&http.Response{StatusCode: http.StatusNotFound}),
				)
				kv.EXPECT().GetSecret(gomock.Any(), "512a50c8-2a43-4c2a-8fd9-a5539475df2a-ingress", "", nil).Times(1).Return(
					azsecrets.GetSecretResponse{Secret: ingress_secret}, nil,
				)
			},
			check: func(i clienthelper.Interface, g Gomega) {
				// check API server certificate is not created, because fetching it errored
				s := &corev1.Secret{}
				err := i.GetOne(ctx, types.NamespacedName{Namespace: "openshift-config", Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-apiserver"}, s)
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())

				s_op := &corev1.Secret{}
				err = i.GetOne(ctx, types.NamespacedName{Namespace: "openshift-azure-operator", Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-apiserver"}, s_op)
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())

				// check ingress server certificate is saved correctly
				err = i.GetOne(ctx, types.NamespacedName{Namespace: "openshift-ingress", Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-ingress"}, s)
				g.Expect(err).ShouldNot(HaveOccurred())

				err = i.GetOne(ctx, types.NamespacedName{Namespace: "openshift-azure-operator", Name: "512a50c8-2a43-4c2a-8fd9-a5539475df2a-ingress"}, s_op)
				g.Expect(err).ShouldNot(HaveOccurred())

				g.Expect(maps.Keys(s.Data)).To(ContainElements(corev1.TLSCertKey, corev1.TLSPrivateKeyKey))
				_, cert, err := utilpem.Parse(s.Data[corev1.TLSCertKey])
				g.Expect(err).ShouldNot(HaveOccurred())

				g.Expect(cert).To(HaveLen(1))
				g.Expect(cert[0].Subject.CommonName).To(Equal("star-cert"))

				// same version is in the operator and openshift-config
				g.Expect(s.Data).To(Equal(s_op.Data))
			},
			objects: []runtime.Object{},
			wantErr: "TransientError: 1 error occurred:\n\t* Request information not available\n--------------------------------------------------------------------------------\nRESPONSE 404: \nERROR CODE UNAVAILABLE\n--------------------------------------------------------------------------------\nResponse contained no body\n--------------------------------------------------------------------------------\n\n\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Domain().AnyTimes().Return("example.com")
			_env.EXPECT().FeatureIsSet(env.FeatureDisableSignedCertificates).AnyTimes().Return(tt.featureDisabled)

			if tt.mocks != nil {
				_kv := mock_azsecrets.NewMockClient(controller)
				tt.mocks(_kv)
				_env.EXPECT().ClusterKeyvault().AnyTimes().Return(_kv)
			}

			_, log := testlog.New()

			builder := fake.NewClientBuilder().WithRuntimeObjects(tt.objects...)
			ch := clienthelper.NewWithClient(log, builder.Build())
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch),
				testtasks.WithOpenShiftClusterProperties(clusterUUID, tt.clusterproperties),
			)

			err := RotateManagedCertificates(tc)
			if tt.wantErr != "" && err != nil {
				g.Expect(err).To(MatchError(tt.wantErr))
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}

			g.Expect(tc.GetResultMessage()).To(Equal(tt.wantMsg))

			if tt.check != nil {
				tt.check(ch, g)
			}
		})
	}
}
