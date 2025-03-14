package cluster

import (
	"context"
	"crypto/x509"
	"errors"
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
	// skip if the cluster is in "Deleting" status.
	if mon.oc.Properties.ProvisioningState == api.ProvisioningStateDeleting {
		return nil
	}

	if err := mon.processCertificate(ctx, operator.Namespace, operator.SecretName, genevalogging.GenevaCertName, nil); err != nil {
		return err
	}
	return nil
}

// emitIngressAndAPIServerCertificateExpiry emits days until expiration for Ingress and API Server certificates.
func (mon *Monitor) emitIngressAndAPIServerCertificateExpiry(ctx context.Context) error {
	host, err := getHostFromAPIURL(mon.oc.Properties.APIServerProfile.URL)
	if err != nil {
		return err
	}

	if dns.IsManagedDomain(host) {
		ic := &operatorv1.IngressController{}
		if err := mon.ocpclientset.Get(ctx, client.ObjectKey{
			Namespace: ingressNamespace,
			Name:      ingressName,
		}, ic); err != nil {
			return err
		}

		if ic.Spec.DefaultCertificate == nil {
			return errors.New("ingress controller spec invalid, default certificate name not found")
		}

		ingressSecretName := ic.Spec.DefaultCertificate.Name
		secretNames := map[string]struct{}{
			ingressSecretName: {},
		}
		apiserverSecretName := strings.Replace(ingressSecretName, "-ingress", "-apiserver", 1)
		secretNames[apiserverSecretName] = struct{}{}

		for secretName := range secretNames {
			if err := mon.processCertificate(ctx, operator.Namespace, secretName, corev1.TLSCertKey, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// emitEtcdCertificateExpiry emits days until expiration for ETCD certificates.
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

	secretList, err := mon.cli.CoreV1().Secrets(etcdNamespace).List(ctx, metav1.ListOptions{
		FieldSelector: "type=" + string(corev1.SecretTypeTLS),
	})
	if err != nil {
		return err
	}

	for _, secret := range secretList.Items {
		secretName := secret.ObjectMeta.Name
		// only process secrets with names indicating ETCD certificates.
		if strings.Contains(secretName, "etcd-peer") || strings.Contains(secretName, "etcd-serving") {
			if err := mon.processCertificate(ctx, etcdNamespace, secretName, corev1.TLSCertKey, &secret); err != nil {
				return err
			}
		}
	}
	return nil
}

// processCertificate is a helper that retrieves a certificate from a secret (or uses the provided secret object),
// calculates days until expiration, and emits a gauge metric.
func (mon *Monitor) processCertificate(ctx context.Context, secretNamespace, secretName, secretKey string, secretObj *corev1.Secret) error {
	// check and get secret if needed
	var cert *x509.Certificate
	var err error
	if secretObj == nil {
		secretObj = &corev1.Secret{}
		err = mon.ocpclientset.Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: secretName}, secretObj)
		if err != nil {
			if kerrors.IsNotFound(err) {
				mon.emitSecretMissingMetric(secretNamespace, secretName)
				return nil
			}
			return err
		}
	}

	certData, ok := secretObj.Data[secretKey]
	if !ok {
		mon.emitSecretMissingMetric(secretObj.Namespace, secretObj.Name)
		return nil
	}

	// get and parse cert with secretObj and secretKey
	cert, err = pem.ParseFirstCertificate(certData)
	if err != nil {
		return err
	}

	// emit the cert expiration metric if the cert is valid
	mon.emitGauge(certificateExpirationMetricName, int64(utilcert.DaysUntilExpiration(cert)), map[string]string{
		"namespace": secretNamespace,
		"name":      secretName,
		"subject":   cert.Subject.CommonName,
	})
	return nil
}

// secretMissingMetric creates a metric label map for a missing secret.
func (mon *Monitor) emitSecretMissingMetric(namespace, name string) {
	secretMissingMetic := map[string]string{
		"namespace": namespace,
		"name":      name,
	}
	mon.emitGauge(secretMissingMetricName, int64(1), secretMissingMetic)
}

// getHostFromAPIURL parses the provided API URL and returns its hostname.
func getHostFromAPIURL(apiURL string) (string, error) {
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Hostname(), nil
}
