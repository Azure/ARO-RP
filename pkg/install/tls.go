package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"encoding/pem"

	configv1 "github.com/openshift/api/config/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
)

func (i *Installer) createCertificates(ctx context.Context) error {
	if _, ok := i.env.(env.Dev); ok {
		return nil
	}

	managedDomain, err := i.env.ManagedDomain(i.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
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
			certificateName: i.doc.ID + "-apiserver",
			commonName:      "api." + managedDomain,
		},
		{
			certificateName: i.doc.ID + "-ingress",
			commonName:      "*.apps." + managedDomain,
		},
	}

	for _, c := range certs {
		i.log.Printf("creating certificate %s", c.certificateName)
		err = i.keyvault.CreateSignedCertificate(ctx, i.env.ClustersKeyvaultURI(), keyvault.IssuerDigicert, c.certificateName, c.commonName, keyvault.EkuServerAuth)
		if err != nil {
			return err
		}
	}

	for _, c := range certs {
		i.log.Printf("waiting for certificate %s", c.certificateName)
		err = i.keyvault.WaitForCertificateOperation(ctx, i.env.ClustersKeyvaultURI(), c.certificateName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) ensureSecret(ctx context.Context, namespace string, certificateName string) error {
	bundle, err := i.keyvault.GetSecret(ctx, i.env.ClustersKeyvaultURI(), certificateName, "")
	if err != nil {
		return err
	}

	key, certs, err := utilpem.Parse([]byte(*bundle.Value))
	if err != nil {
		return err
	}

	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}

	var cb []byte
	for _, cert := range certs {
		cb = append(cb, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})...)
	}

	err = i.adminactions.ApplySecret(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certificateName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			v1.TLSCertKey:       cb,
			v1.TLSPrivateKeyKey: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
		},
		Type: v1.SecretTypeTLS,
	})
	return err
}

func (i *Installer) configureAPIServerCertificate(ctx context.Context) error {
	if _, ok := i.env.(env.Dev); ok {
		return nil
	}

	managedDomain, err := i.env.ManagedDomain(i.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if managedDomain == "" {
		return nil
	}

	err = i.ensureSecret(ctx, "openshift-config", i.doc.ID+"-apiserver")
	if err != nil {
		return err
	}

	return i.adminactions.ApplyAPIServerNamedServingCert(&configv1.APIServerNamedServingCert{
		Names: []string{
			"api." + managedDomain,
		},
		ServingCertificate: configv1.SecretNameReference{
			Name: i.doc.ID + "-apiserver",
		},
	})
}

func (i *Installer) configureIngressCertificate(ctx context.Context) error {
	if _, ok := i.env.(env.Dev); ok {
		return nil
	}

	managedDomain, err := i.env.ManagedDomain(i.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if managedDomain == "" {
		return nil
	}

	err = i.ensureSecret(ctx, "openshift-ingress", i.doc.ID+"-ingress")
	if err != nil {
		return err
	}

	return i.adminactions.ApplyIngressControllerCertificate(&v1.LocalObjectReference{
		Name: i.doc.ID + "-ingress",
	})
}
