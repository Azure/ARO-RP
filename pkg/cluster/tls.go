package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
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
			certificateName: m.doc.ID + "-apiserver",
			commonName:      "api." + managedDomain,
		},
		{
			certificateName: m.doc.ID + "-ingress",
			commonName:      "*.apps." + managedDomain,
		},
	}

	for _, c := range certs {
		m.log.Printf("creating certificate %s", c.certificateName)
		err = m.env.ClusterKeyvault().CreateSignedCertificate(ctx, "OneCertV2-PublicCA", c.certificateName, c.commonName, keyvault.EkuServerAuth)
		if err != nil {
			return err
		}
	}

	for _, c := range certs {
		m.log.Printf("waiting for certificate %s", c.certificateName)
		err = m.env.ClusterKeyvault().WaitForCertificateOperation(ctx, c.certificateName)
		if err != nil {
			m.log.Errorf("error when waiting for certificate %s: %s", c.certificateName, err.Error())
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
		err = EnsureTLSSecretFromKeyvault(ctx, m.env, m.ch, types.NamespacedName{Name: m.doc.ID + "-apiserver", Namespace: namespace}, m.doc.ID+"-apiserver")
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
					Name: m.doc.ID + "-apiserver",
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
		err = EnsureTLSSecretFromKeyvault(ctx, m.env, m.ch, types.NamespacedName{Namespace: namespace, Name: m.doc.ID + "-ingress"}, m.doc.ID+"-ingress")
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
			Name: m.doc.ID + "-ingress",
		}

		_, err = m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Update(ctx, ic, metav1.UpdateOptions{})
		return err
	})
}
