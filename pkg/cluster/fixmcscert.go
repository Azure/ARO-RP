package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net"
	"strings"

	"github.com/openshift/installer/pkg/asset/tls"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) fixMCSCert(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	intIP := net.ParseIP(m.doc.OpenShiftCluster.Properties.APIServerProfile.IntIP)

	domain := m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(domain, '.') {
		domain += "." + m.env.Domain()
	}

	var rootCA *tls.RootCA
	var certChanged bool

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		s, err := m.kubernetescli.CoreV1().Secrets("openshift-machine-config-operator").Get(ctx, "machine-config-server-tls", metav1.GetOptions{})
		if err != nil {
			return err
		}

		_, certs, err := utilpem.Parse(s.Data[corev1.TLSCertKey])
		if err != nil {
			return err
		}

		if len(certs) != 1 {
			return fmt.Errorf("expected 1 certificate, got %d", len(certs))
		}

		if len(certs[0].IPAddresses) == 1 && certs[0].IPAddresses[0].Equal(intIP) {
			return nil
		}

		certChanged = true

		if rootCA == nil {
			pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
			if err != nil {
				return err
			}

			err = pg.Get(true, &rootCA)
			if err != nil {
				return err
			}
		}

		cfg := &tls.CertCfg{
			Subject:      pkix.Name{CommonName: "system:machine-config-server"},
			ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			Validity:     tls.ValidityTenYears,
			IPAddresses:  []net.IP{intIP},
			DNSNames:     []string{"api-int." + domain, intIP.String()},
		}

		var mcsCertKey tls.AdminKubeConfigClientCertKey

		err = mcsCertKey.SignedCertKey.Generate(cfg, rootCA, "machine-config-server", tls.DoNotAppendParent)
		if err != nil {
			return err
		}

		s.Data[corev1.TLSCertKey] = mcsCertKey.CertRaw
		s.Data[corev1.TLSPrivateKeyKey] = mcsCertKey.KeyRaw

		_, err = m.kubernetescli.CoreV1().Secrets("openshift-machine-config-operator").Update(ctx, s, metav1.UpdateOptions{})
		return err
	})
	if err != nil || !certChanged {
		return err
	}

	/* don't crash */

	return m.kubernetescli.CoreV1().Pods("openshift-machine-config-operator").DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: "k8s-app=machine-config-server",
	})
}
