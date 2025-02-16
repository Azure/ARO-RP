package cluster

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/api"
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
	etcdNamespace                   = "openshift-etcd"
)

// emitMDSDCertificateExpiry emits days until expiration for the Geneva logging (MDSD) certificate.
func (mon *Monitor) emitMDSDCertificateExpiry(ctx context.Context) error {
	// Skip if the cluster is in "Deleting" status.
	if mon.oc.Properties.ProvisioningState == api.ProvisioningStateDeleting {
		return nil
	}

	// Retrieve and check the MDSD certificate via helper function.
	if err := mon.processCertificate(ctx, operator.Namespace, operator.SecretName, genevalogging.GenevaCertName); err != nil {
		// If the secret is not found, emit a missing secret metric.
		if kerrors.IsNotFound(err) {
			mon.emitGauge(secretMissingMetricName, 1, secretMissingMetric(operator.Namespace, operator.SecretName))
			return nil
		}
		return fmt.Errorf("emitting MDSD certificate expiration metrics: %w", err)
	}
	return nil
}

// emitIngressAndAPIServerCertificateExpiry emits days until expiration for Ingress and API Server certificates.
func (mon *Monitor) emitIngressAndAPIServerCertificateExpiry(ctx context.Context) error {
	host, err := getHostFromAPIURL(mon.oc.Properties.APIServerProfile.URL)
	if err != nil {
		return fmt.Errorf("parsing API URL: %w", err)
	}

	if dns.IsManagedDomain(host) {
		ic := &operatorv1.IngressController{}
		if err := mon.ocpclientset.Get(ctx, client.ObjectKey{
			Namespace: ingressNamespace,
			Name:      ingressName,
		}, ic); err != nil {
			return fmt.Errorf("getting ingress controller %s/%s: %w", ingressNamespace, ingressName, err)
		}

		if ic.Spec.DefaultCertificate == nil {
			return fmt.Errorf("ingress controller spec invalid, default certificate name not found")
		}

		ingressSecretName := ic.Spec.DefaultCertificate.Name
		// Build a set of secret names to process, avoid duplicate processing.
		secretNames := map[string]struct{}{
			ingressSecretName: {},
		}
		// Also process the API Server certificate variant if it differs.
		apiserverSecretName := strings.Replace(ingressSecretName, "-ingress", "-apiserver", 1)
		secretNames[apiserverSecretName] = struct{}{}

		for secretName := range secretNames {
			if err := mon.processCertificate(ctx, operator.Namespace, secretName, corev1.TLSCertKey); err != nil {
				if kerrors.IsNotFound(err) {
					mon.emitGauge(secretMissingMetricName, 1, secretMissingMetric(operator.Namespace, secretName))
				} else {
					return fmt.Errorf("emitting %q expiration metrics: %w", secretName, err)
				}
			}
		}
	}

	return nil
}

// emitEtcdCertificateExpiry emits days until expiration for ETCD certificates.
// Note: For clusters running version 4.9 or higher, ETCD certificates are auto-rotated.
func (mon *Monitor) emitEtcdCertificateExpiry(ctx context.Context) error {
	cv, err := mon.getClusterVersion(ctx)
	if err != nil {
		return fmt.Errorf("getting cluster version: %w", err)
	}
	v, err := version.ParseVersion(actualVersion(cv))
	if err != nil {
		return fmt.Errorf("parsing cluster version: %w", err)
	}
	// Only process ETCD certificates for clusters running a version less than 4.9.
	if !v.Lt(version.NewVersion(4, 9)) {
		return nil
	}

	secretList, err := mon.cli.CoreV1().Secrets(etcdNamespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("type=%s", corev1.SecretTypeTLS),
	})
	if err != nil {
		return fmt.Errorf("listing secrets in %q: %w", etcdNamespace, err)
	}

	for _, secret := range secretList.Items {
		secretName := secret.ObjectMeta.Name
		// Only process secrets with names indicating ETCD certificates.
		if strings.Contains(secretName, "etcd-peer") || strings.Contains(secretName, "etcd-serving") {
			certData, ok := secret.Data[corev1.TLSCertKey]
			if !ok {
				mon.emitGauge(secretMissingMetricName, 1, secretMissingMetric(etcdNamespace, secretName))
				continue
			}
			_, certs, err := pem.Parse(certData)
			if err != nil {
				// Log the error and continue processing other secrets.
				mon.log.Printf("warning: parsing certificate in secret %q: %v", secretName, err)
				continue
			}
			if len(certs) == 0 {
				mon.log.Printf("warning: no certificates found in secret %q", secretName)
				continue
			}
			mon.emitGauge(certificateExpirationMetricName, int64(utilcert.DaysUntilExpiration(certs[0])), map[string]string{
				"namespace": etcdNamespace,
				"name":      secretName,
				"subject":   certs[0].Subject.CommonName,
			})
		}
	}

	return nil
}

// processCertificate is a helper that retrieves a certificate from a secret,
// calculates days until expiration, and emits a gauge metric.
func (mon *Monitor) processCertificate(ctx context.Context, secretNamespace, secretName, secretKey string) error {
	cert, err := mon.getCertificate(ctx, secretNamespace, secretName, secretKey)
	if err != nil {
		return err
	}
	mon.emitGauge(certificateExpirationMetricName, int64(utilcert.DaysUntilExpiration(cert)), map[string]string{
		"namespace": secretNamespace,
		"name":      secretName,
		"subject":   cert.Subject.CommonName,
	})
	return nil
}

// getCertificate retrieves a certificate from the specified secret and key.
func (mon *Monitor) getCertificate(ctx context.Context, secretNamespace, secretName, secretKey string) (*x509.Certificate, error) {
	secret := &corev1.Secret{}
	if err := mon.ocpclientset.Get(ctx, client.ObjectKey{
		Namespace: secretNamespace,
		Name:      secretName,
	}, secret); err != nil {
		return nil, fmt.Errorf("getting secret %q in namespace %q: %w", secretName, secretNamespace, err)
	}
	certData, ok := secret.Data[secretKey]
	if !ok {
		return nil, fmt.Errorf("secret %q in namespace %q does not contain key %q", secretName, secretNamespace, secretKey)
	}
	cert, err := pem.ParseFirstCertificate(certData)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate from secret %q in namespace %q: %w", secretName, secretNamespace, err)
	}
	return cert, nil
}

// secretMissingMetric creates a metric label map for a missing secret.
func secretMissingMetric(namespace, name string) map[string]string {
	return map[string]string{
		"namespace": namespace,
		"name":      name,
	}
}

// getHostFromAPIURL parses the provided API URL and returns its hostname.
func getHostFromAPIURL(apiURL string) (string, error) {
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("parsing API URL %q: %w", apiURL, err)
	}
	return parsedURL.Hostname(), nil
}
