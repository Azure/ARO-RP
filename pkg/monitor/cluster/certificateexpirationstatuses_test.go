package cluster

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	operatorv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
type certInfo struct {
	secretName, certSubject string
}

const (
	managedDomainName   = "contoso.aroapp.io"
	unmanagedDomainName = "aro.contoso.com"
)

func TestEmitCertificateExpirationStatuses(t *testing.T) {
	expiration := time.Now().Add(time.Hour * 24 * 5)
	expirationString := expiration.UTC().Format(time.RFC3339)
	clusterID := uuid.DefaultGenerator.Generate()

	for _, tt := range []struct {
		name            string
		domain          string
		certsPresent    []certInfo
		wantExpirations []map[string]string
		wantWarning     []map[string]string
		wantErr         string
	}{
		{
			name:         "only emits MDSD status for unmanaged domain",
			domain:       unmanagedDomainName,
			certsPresent: []certInfo{{"cluster", "geneva.certificate"}},
			wantExpirations: []map[string]string{
				{
					"subject":             "geneva.certificate",
					"expirationDate":      expirationString,
					"daysUntilExpiration": "5",
				},
			},
		},
		{
			name:   "includes ingress and API status for managed domain",
			domain: managedDomainName,
			certsPresent: []certInfo{
				{"cluster", "geneva.certificate"},
				{clusterID + "-ingress", managedDomainName},
				{clusterID + "-apiserver", "api." + managedDomainName},
			},
			wantExpirations: []map[string]string{
				{
					"subject":             "geneva.certificate",
					"expirationDate":      expirationString,
					"daysUntilExpiration": "5",
				},
				{
					"subject":             "contoso.aroapp.io",
					"expirationDate":      expirationString,
					"daysUntilExpiration": "5",
				},
				{
					"subject":             "api.contoso.aroapp.io",
					"expirationDate":      expirationString,
					"daysUntilExpiration": "5",
				},
			},
		},
		{
			name:   "emits warning metric when cluster secret has been deleted",
			domain: unmanagedDomainName,
			wantWarning: []map[string]string{
				{
					"namespace": "openshift-azure-operator",
					"name":      "cluster",
				},
			},
		},
		{
			name:   "emits warning metric when managed domain secret has been deleted",
			domain: managedDomainName,
			certsPresent: []certInfo{
				{"cluster", "geneva.certificate"},
				{clusterID + "-ingress", managedDomainName},
			},
			wantExpirations: []map[string]string{
				{
					"subject":             "geneva.certificate",
					"expirationDate":      expirationString,
					"daysUntilExpiration": "5",
				},
				{
					"subject":             "contoso.aroapp.io",
					"expirationDate":      expirationString,
					"daysUntilExpiration": "5",
				},
			},
			wantWarning: []map[string]string{
				{
					"namespace": "openshift-azure-operator",
					"name":      clusterID + "-apiserver",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var secrets []client.Object
			secretsFromCertInfo, err := generateTestSecrets(tt.certsPresent, tweakTemplateFn(expiration))
			if err != nil {
				t.Fatal(err)
			}
			secrets = append(secrets, secretsFromCertInfo...)

			m := mock_metrics.NewMockEmitter(gomock.NewController(t))
			for _, w := range tt.wantWarning {
				m.EXPECT().EmitGauge(secretMissingMetricName, int64(1), w)
			}
			for _, g := range tt.wantExpirations {
				m.EXPECT().EmitGauge(certificateExpirationMetricName, int64(1), g)
			}

			mon := buildMonitor(m, tt.domain, clusterID, secrets...)

			err = mon.emitCertificateExpirationStatuses(ctx)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}

	t.Run("returns error when secret is present but certificate data has been deleted", func(t *testing.T) {
		var secrets []client.Object
		data := map[string][]byte{}
		s := buildSecret("cluster", data)
		secrets = append(secrets, s)

		ctx := context.Background()
		m := mock_metrics.NewMockEmitter(gomock.NewController(t))
		mon := buildMonitor(m, managedDomainName, clusterID, secrets...)

		wantErr := `certificate "gcscert.pem" not found on secret "cluster"`
		err := mon.emitCertificateExpirationStatuses(ctx)
		utilerror.AssertErrorMessage(t, err, wantErr)
	})
}

func tweakTemplateFn(expiration time.Time) func(*x509.Certificate) {
	return func(template *x509.Certificate) {
		template.NotAfter = expiration
	}
}

func generateTestSecrets(certsInfo []certInfo, tweakTemplateFn func(*x509.Certificate)) ([]client.Object, error) {
	var secrets []client.Object
	for _, sec := range certsInfo {
		_, cert, err := utiltls.GenerateTestKeyAndCertificate(sec.certSubject, nil, nil, false, false, tweakTemplateFn)
		if err != nil {
			return nil, err
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
	}
	return secrets, nil
}

func buildSecret(secretName string, data map[string][]byte) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "openshift-azure-operator",
		},
		Data: data,
	}
	return s
}

func buildMonitor(m *mock_metrics.MockEmitter, domain, id string, secrets ...client.Object) *Monitor {
	ingressController := &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "openshift-ingress-operator",
		},
		Spec: operatorv1.IngressControllerSpec{
			DefaultCertificate: &corev1.LocalObjectReference{
				Name: id + "-ingress",
			},
		},
	}

	clientset := fake.
		NewClientBuilder().
		WithObjects(ingressController).
		WithObjects(secrets...).
		Build()
	mon := &Monitor{
		clientset: clientset,
		m:         m,
		oc: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					Domain: domain,
				},
			},
		},
	}
	return mon
}
