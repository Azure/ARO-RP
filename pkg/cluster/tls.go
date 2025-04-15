package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	azcertificates_sdk "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcertificates"
	"github.com/Azure/ARO-RP/pkg/util/dns"
)

const (
	OneCertPublicIssuerName = "OneCertV2-PublicCA"
)

func (m *manager) createCertificates(ctx context.Context) error {
	if m.env.FeatureIsSet(env.FeatureDisableSignedCertificates) {
		return nil
	}

	managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if managedDomain == "" {
		return nil
	}

	certs := []struct {
		certificateName string
		commonName      string
	}{
		{
			certificateName: m.APICertName(),
			commonName:      "api." + managedDomain,
		},
		{
			certificateName: m.IngressCertName(),
			commonName:      "*.apps." + managedDomain,
		},
	}

	for _, c := range certs {
		m.log.Printf("creating certificate %s", c.certificateName)
		_, err = m.env.ClusterCertificates().CreateCertificate(ctx, c.certificateName, azcertificates.SignedCertificateParameters(OneCertPublicIssuerName, c.commonName, azcertificates.EkuServerAuth), nil)
		if err != nil {
			return err
		}
	}

	for _, c := range certs {
		m.log.Printf("waiting for certificate %s", c.certificateName)
		if err := azcertificates.WaitForCertificateOperation(ctx, m.log, func(ctx context.Context) (azcertificates_sdk.CertificateOperation, error) {
			op, err := m.env.ClusterCertificates().GetCertificateOperation(ctx, c.certificateName, nil)
			if err != nil {
				return azcertificates_sdk.CertificateOperation{}, err
			}
			return op.CertificateOperation, err
		}); err != nil {
			m.log.Errorf("error when getting operation for certificate %s: %s", c.certificateName, err.Error())
			return err
		}
	}

	return nil
}

func (m *manager) configureAPIServerCertificate(ctx context.Context) error {
	if m.env.FeatureIsSet(env.FeatureDisableSignedCertificates) {
		return nil
	}

	managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if managedDomain == "" {
		return nil
	}

	for _, namespace := range []string{"openshift-config", "openshift-azure-operator"} {
		err = EnsureTLSSecretFromKeyvault(ctx, m.env.ClusterKeyvault(), m.ch, types.NamespacedName{Name: m.APICertName(), Namespace: namespace}, m.APICertName())
		if err != nil {
			return err
		}
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		apiserver, err := m.configcli.ConfigV1().APIServers().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		apiserver.Spec.ServingCerts.NamedCertificates = []configv1.APIServerNamedServingCert{
			{
				Names: []string{
					"api." + managedDomain,
				},
				ServingCertificate: configv1.SecretNameReference{
					Name: m.APICertName(),
				},
			},
		}

		_, err = m.configcli.ConfigV1().APIServers().Update(ctx, apiserver, metav1.UpdateOptions{})
		return err
	})
}

func (m *manager) configureIngressCertificate(ctx context.Context) error {
	if m.env.FeatureIsSet(env.FeatureDisableSignedCertificates) {
		return nil
	}

	managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if managedDomain == "" {
		return nil
	}

	for _, namespace := range []string{"openshift-ingress", "openshift-azure-operator"} {
		err = EnsureTLSSecretFromKeyvault(ctx, m.env.ClusterKeyvault(), m.ch, types.NamespacedName{Namespace: namespace, Name: m.IngressCertName()}, m.IngressCertName())
		if err != nil {
			return err
		}
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ic, err := m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		ic.Spec.DefaultCertificate = &corev1.LocalObjectReference{
			Name: m.IngressCertName(),
		}

		_, err = m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Update(ctx, ic, metav1.UpdateOptions{})
		return err
	})
}
