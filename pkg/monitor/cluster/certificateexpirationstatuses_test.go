package cluster

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeClient "k8s.io/client-go/kubernetes/fake"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	utilcert "github.com/Azure/ARO-RP/pkg/util/cert"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
type certInfo struct {
	secretName, certSubject string
}

const (
	managedDomainName     = "contoso.aroapp.io"
	unmanagedDomainName   = "aro.contoso.com"
	managedDomainApiURL   = "https://api.contoso.aroapp.io:6443"
	unmanagedDomainApiURL = "https://api.aro.contoso.com:6443"
)

func TestEmitMDSDCertificateExpiry(t *testing.T) {
	expiration := time.Now().Add(time.Hour * 24 * 5)
	daysUntilExpiration := 4

	for _, tt := range []struct {
		name            string
		clusterState    api.ProvisioningState
		certsPresent    []certInfo
		wantExpirations []map[string]string
		wantWarning     []map[string]string
		wantErr         string
	}{
		{
			name:            "cluster is in deleting status",
			clusterState:    api.ProvisioningStateDeleting,
			certsPresent:    []certInfo{{"cluster", "geneva.certificate"}},
			wantExpirations: []map[string]string{},
		},
		{
			name:            "secret not found",
			clusterState:    api.ProvisioningStateSucceeded,
			certsPresent:    []certInfo{},
			wantExpirations: []map[string]string{},
			wantWarning: []map[string]string{
				{
					"namespace": "openshift-azure-operator",
					"name":      "cluster",
				},
			},
		},
		{
			name:         "emit MDSD cert status",
			clusterState: api.ProvisioningStateSucceeded,
			certsPresent: []certInfo{{"cluster", "geneva.certificate"}},
			wantExpirations: []map[string]string{
				{
					"subject":   "geneva.certificate",
					"name":      "cluster",
					"namespace": "openshift-azure-operator",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var secrets []client.Object
			secretsFromCertInfo, thumbprints, err := generateTestSecrets(tt.certsPresent, tweakTemplateFn(expiration))
			if err != nil {
				t.Fatal(err)
			}
			secrets = append(secrets, secretsFromCertInfo...)

			m := mock_metrics.NewMockEmitter(gomock.NewController(t))

			_, log := testlog.New()
			ocpclientset := clienthelper.NewWithClient(log, fake.
				NewClientBuilder().
				WithObjects(secrets...).
				Build())

			mon := &Monitor{
				ocpclientset: ocpclientset,
				m:            m,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: tt.clusterState,
					},
				},
			}

			for _, warning := range tt.wantWarning {
				m.EXPECT().EmitGauge(secretMissingMetricName, int64(1), warning)
			}

			for i, exp := range tt.wantExpirations {
				// add thumbprint to expected dimensions, since the certificates are dynamically generated we can not define these in the test cases themselves
				exp["thumbprint"] = thumbprints[i]

				m.EXPECT().EmitGauge(certificateExpirationMetricName, int64(daysUntilExpiration), exp)
			}

			err = mon.emitMDSDCertificateExpiry(ctx)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestEmitIngressAndAPIServerCertificateExpiry(t *testing.T) {
	expiration := time.Now().Add(time.Hour * 24 * 5)
	daysUntilExpiration := 4
	clusterID := "00000000-0000-0000-0000-000000000000"
	defaultIngressController := &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "openshift-ingress-operator",
		},
		Spec: operatorv1.IngressControllerSpec{
			DefaultCertificate: &corev1.LocalObjectReference{
				Name: clusterID + "-ingress",
			},
		},
	}

	for _, tt := range []struct {
		name              string
		url               string
		ingressController *operatorv1.IngressController
		certsPresent      []certInfo
		wantExpirations   []map[string]string
		wantWarning       []map[string]string
		wantErr           string
	}{
		{
			name:              "emit ingress and API status for managed domain",
			url:               managedDomainApiURL,
			ingressController: defaultIngressController,
			certsPresent: []certInfo{
				{clusterID + "-ingress", managedDomainName},
				{clusterID + "-apiserver", "api." + managedDomainName},
			},
			wantExpirations: []map[string]string{
				{
					"subject":   "contoso.aroapp.io",
					"name":      clusterID + "-ingress",
					"namespace": "openshift-azure-operator",
				},
				{
					"subject":   "api.contoso.aroapp.io",
					"name":      clusterID + "-apiserver",
					"namespace": "openshift-azure-operator",
				},
			},
		},
		{
			name:              "not emit ingress and API status for unmanaged domain",
			url:               unmanagedDomainApiURL,
			ingressController: defaultIngressController,
			certsPresent: []certInfo{
				{clusterID + "-ingress", unmanagedDomainName},
				{clusterID + "-apiserver", "api." + unmanagedDomainName},
			},
			wantExpirations: []map[string]string{},
		},
		{
			name:              "emits warning metric when managed domain secret has been deleted",
			url:               managedDomainApiURL,
			ingressController: defaultIngressController,
			certsPresent: []certInfo{
				{clusterID + "-ingress", managedDomainName},
			},
			wantExpirations: []map[string]string{
				{
					"namespace": "openshift-azure-operator",
					"subject":   "contoso.aroapp.io",
					"name":      clusterID + "-ingress",
				},
			},
			wantWarning: []map[string]string{
				{
					"namespace": "openshift-azure-operator",
					"name":      clusterID + "-apiserver",
				},
			},
		},
		{
			name: "returns error and does not panic when managed domain cluster has invalid ingresscontroller resource",
			url:  managedDomainApiURL,
			ingressController: &operatorv1.IngressController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "openshift-ingress-operator",
				},
				Spec: operatorv1.IngressControllerSpec{},
			},
			certsPresent:    []certInfo{},
			wantExpirations: []map[string]string{},
			wantErr:         "ingress controller spec invalid, default certificate name not found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var secrets []client.Object
			secretsFromCertInfo, thumbprints, err := generateTestSecrets(tt.certsPresent, tweakTemplateFn(expiration))
			if err != nil {
				t.Fatal(err)
			}
			secrets = append(secrets, secretsFromCertInfo...)

			m := mock_metrics.NewMockEmitter(gomock.NewController(t))

			_, log := testlog.New()
			ocpclientset := clienthelper.NewWithClient(log, fake.
				NewClientBuilder().
				WithObjects(tt.ingressController).
				WithObjects(secrets...).
				Build())

			mon := &Monitor{
				ocpclientset: ocpclientset,
				m:            m,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						APIServerProfile: api.APIServerProfile{
							URL: tt.url,
						},
					},
				},
			}

			for _, warning := range tt.wantWarning {
				m.EXPECT().EmitGauge(secretMissingMetricName, int64(1), warning)
			}

			for i, exp := range tt.wantExpirations {
				// add thumbprint to expected dimensions, since the certificates are dynamically generated we can not define these in the test cases themselves
				exp["thumbprint"] = thumbprints[i]

				m.EXPECT().EmitGauge(certificateExpirationMetricName, int64(daysUntilExpiration), exp)
			}

			err = mon.emitIngressAndAPIServerCertificateExpiry(ctx)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestEtcdCertificateExpiry(t *testing.T) {
	ctx := context.Background()
	expiration := time.Now().Add(time.Microsecond * 60)
	_, certificate, err := utiltls.GenerateTestKeyAndCertificate("etcd-cert", nil, nil, false, false, tweakTemplateFn(expiration))
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name                   string
		configcli              *configfake.Clientset
		cli                    *fakeClient.Clientset
		minDaysUntilExpiration int
	}{
		{
			name: "emit etcd certificate expiry",
			configcli: configfake.NewSimpleClientset(
				&configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Status: configv1.ClusterVersionStatus{
						History: []configv1.UpdateHistory{
							{
								State:   configv1.CompletedUpdate,
								Version: "4.8.1",
							},
						},
					},
				},
			),
			cli: fakeClient.NewSimpleClientset(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "etcd-peer-master-0",
						Namespace: "openshift-etcd",
					},
					Data: map[string][]byte{
						corev1.TLSCertKey: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate[0].Raw}),
					},
					Type: corev1.SecretTypeTLS,
				},
			),
			minDaysUntilExpiration: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)
			mon := &Monitor{
				cli:       tt.cli,
				configcli: tt.configcli,
				m:         m,
			}

			m.EXPECT().EmitGauge(certificateExpirationMetricName, int64(tt.minDaysUntilExpiration), map[string]string{
				"namespace":  "openshift-etcd",
				"name":       "etcd-peer-master-0",
				"subject":    "etcd-cert",
				"thumbprint": utilcert.Thumbprint(certificate[0]),
			})
			err = mon.emitEtcdCertificateExpiry(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func tweakTemplateFn(expiration time.Time) func(*x509.Certificate) {
	return func(template *x509.Certificate) {
		template.NotAfter = expiration
	}
}

func generateTestSecrets(certsInfo []certInfo, tweakTemplateFn func(*x509.Certificate)) ([]client.Object, []string, error) {
	var secrets []client.Object
	var thumbprints []string
	for _, sec := range certsInfo {
		_, cert, err := utiltls.GenerateTestKeyAndCertificate(sec.certSubject, nil, nil, false, false, tweakTemplateFn)
		if err != nil {
			return nil, nil, err
		}
		certKey := "tls.crt"
		if sec.secretName == "cluster" {
			certKey = "gcscert.pem"
		}
		data := map[string][]byte{
			certKey: pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert[0].Raw,
			}),
		}
		s := buildSecret(sec.secretName, data)
		secrets = append(secrets, s)
		thumbprints = append(thumbprints, utilcert.Thumbprint(cert[0]))
	}
	return secrets, thumbprints, nil
}

func buildSecret(secretName string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "openshift-azure-operator",
		},
		Data: data,
	}
}
