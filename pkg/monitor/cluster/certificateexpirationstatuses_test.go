package cluster

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestEmitCertificateExpirationStatuses(t *testing.T) {
	expiration := time.Now().Add(time.Hour * 24 * 5)
	expirationString := expiration.UTC().Format(time.RFC3339)
	for _, tt := range []struct {
		name            string
		isManaged       bool
		wantExpirations []map[string]string
	}{
		{
			name:      "unmanaged domain",
			isManaged: false,
			wantExpirations: []map[string]string{
				{
					"subject":        "geneva.certificate",
					"expirationDate": expirationString,
				},
			},
		},
		{
			name:      "managed domain",
			isManaged: true,
			wantExpirations: []map[string]string{
				{
					"subject":        "geneva.certificate",
					"expirationDate": expirationString,
				},
				{
					"subject":        "contoso.aroapp.io",
					"expirationDate": expirationString,
				},
				{
					"subject":        "api.contoso.aroapp.io",
					"expirationDate": expirationString,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var secrets []runtime.Object
			_, genevaCert, err := utiltls.GenerateTestKeyAndCertificate("geneva.certificate", nil, nil, false, false, func(template *x509.Certificate) {
				template.NotAfter = expiration
			})
			secrets = append(secrets, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster",
					Namespace: "openshift-azure-operator",
				},
				Data: map[string][]byte{
					"gcscert.pem": pem.EncodeToMemory(&pem.Block{
						Type:  "CERTIFICATE",
						Bytes: genevaCert[0].Raw,
					}),
				},
			})

			var domain string
			if tt.isManaged {
				domain = "contoso.aroapp.io"
				for _, sec := range []struct{ secretName, certSubject string }{
					{
						"foo12-ingress",
						domain,
					},
					{
						"foo12-apiserver",
						"api." + domain,
					},
				} {
					_, cert, _ := utiltls.GenerateTestKeyAndCertificate(sec.certSubject, nil, nil, false, false, func(template *x509.Certificate) {
						template.NotAfter = expiration
					})
					secrets = append(secrets, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      sec.secretName,
							Namespace: "openshift-azure-operator",
						},
						Data: map[string][]byte{
							"tls.crt": pem.EncodeToMemory(&pem.Block{
								Type:  "CERTIFICATE",
								Bytes: cert[0].Raw,
							}),
						},
					})
				}
			} else {
				domain = "aro.contoso.com"
			}

			m := mock_metrics.NewMockEmitter(gomock.NewController(t))
			for _, gauge := range tt.wantExpirations {
				m.EXPECT().EmitGauge("certificate.expirationdate", int64(1), gauge)
			}
			m.EXPECT().EmitGauge("managedCertificates.count", int64(len(tt.wantExpirations)), map[string]string{})

			mon := &Monitor{
				cli: fake.NewSimpleClientset(secrets...),
				m:   m,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Domain: domain,
						},
						InfraID: "foo12",
					},
				},
			}

			err = mon.emitCertificateExpirationStatuses(ctx)
			if err != nil {
				t.Errorf("got error %v", err)
			}
		})
	}
}
