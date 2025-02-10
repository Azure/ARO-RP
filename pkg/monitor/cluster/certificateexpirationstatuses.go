package cluster

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	utilcert "github.com/Azure/ARO-RP/pkg/util/cert"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
const (
	certificateExpirationMetricName = "certificate.expirationdate"
	secretMissingMetricName         = "certificate.secretnotfound"
	ingressNamespace                = "openshift-ingress-operator"
	ingressName                     = "default"
)

// report NotAfter dates for Ingress and API (on managed domains), and Geneva (always)
func (mon *Monitor) emitCertificateExpirationStatuses(ctx context.Context) error {
	mdsdCert, err := mon.getCertificate(ctx, operator.Namespace, operator.SecretName, genevalogging.GenevaCertName)
	if kerrors.IsNotFound(err) {
		mon.emitGauge(secretMissingMetricName, int64(1), secretMissingMetric(operator.Namespace, operator.SecretName))
	} else if err != nil {
		return err
	} else {
		daysUntilExpiration := time.Until(mdsdCert.NotAfter) / (24 * time.Hour)
		mon.emitGauge(certificateExpirationMetricName, int64(daysUntilExpiration), map[string]string{
			"subject":   mdsdCert.Subject.CommonName,
			"name":      operator.SecretName,
			"namespace": operator.Namespace,
		})
	}

	host, err := getHostFromAPIURL(mon.oc.Properties.APIServerProfile.URL)
	if err != nil {
		return err
	}

	if dns.IsManagedDomain(host) {
		ic := &operatorv1.IngressController{}
		err := mon.ocpclientset.Get(ctx, client.ObjectKey{
			Namespace: ingressNamespace,
			Name:      ingressName,
		}, ic)
		if err != nil {
			return err
		}

		if ic.Spec.DefaultCertificate == nil {
			return fmt.Errorf("ingresscontroller spec invalid, unable to get default certificate name")
		}

		ingressSecretName := ic.Spec.DefaultCertificate.Name

		// secret with managed certificates is uuid + "-ingress" or "-apiserver"
		for _, secretName := range []string{ingressSecretName, strings.Replace(ingressSecretName, "-ingress", "-apiserver", 1)} {
			certificate, err := mon.getCertificate(ctx, operator.Namespace, secretName, corev1.TLSCertKey)
			if kerrors.IsNotFound(err) {
				mon.emitGauge(secretMissingMetricName, int64(1), secretMissingMetric(operator.Namespace, secretName))
			} else if err != nil {
				return err
			} else {
				daysUntilExpiration := time.Until(certificate.NotAfter) / (24 * time.Hour)
				mon.emitGauge(certificateExpirationMetricName, int64(daysUntilExpiration), map[string]string{
					"subject":   certificate.Subject.CommonName,
					"name":      secretName,
					"namespace": operator.Namespace,
				})
			}
		}
	}

	return nil
}

func (mon *Monitor) getCertificate(ctx context.Context, secretNamespace, secretName, secretKey string) (*x509.Certificate, error) {
	secret := &corev1.Secret{}
	err := mon.ocpclientset.Get(ctx, client.ObjectKey{
		Namespace: secretNamespace,
		Name:      secretName,
	}, secret)
	if err != nil {
		return nil, err
	}

	return pem.ParseFirstCertificate(secret.Data[secretKey])
}

func secretMissingMetric(namespace, name string) map[string]string {
	return map[string]string{
		"namespace": namespace,
		"name":      name,
	}
}

func getHostFromAPIURL(apiURL string) (string, error) {
	domain, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	return domain.Hostname(), nil
}

func (mon *Monitor) emitEtcdCertificateExpiry(ctx context.Context) error {
	cv, err := mon.getClusterVersion(ctx)
	if err != nil {
		return err
	}
	v, err := version.ParseVersion(actualVersion(cv))
	if err != nil {
		return err
	}
	// ETCD ceritificates are autorotated by the operator when close to expiry for cluster running 4.9+
	if !v.Lt(version.NewVersion(4, 9)) {
		return nil
	}

	secretList, err := mon.cli.CoreV1().Secrets("openshift-etcd").List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("type=%s", corev1.SecretTypeTLS)})
	if err != nil {
		return err
	}

	for _, secret := range secretList.Items {
		// Parse secrets with name containing "etcd-peer", "etcd-serving", "etcd-serving-metrics"
		if strings.Contains(secret.ObjectMeta.Name, "etcd-peer") || strings.Contains(secret.ObjectMeta.Name, "etcd-serving") {
			_, certs, err := pem.Parse(secret.Data[corev1.TLSCertKey])
			if err != nil {
				return err
			}
			mon.emitGauge(certificateExpirationMetricName, int64(utilcert.DaysUntilExpiration(certs[0])), map[string]string{
				"namespace": "openshift-etcd",
				"name":      secret.GetObjectMeta().GetName(),
				"subject":   certs[0].Subject.CommonName,
			})
		}
	}

	return nil
}
