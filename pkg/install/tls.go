package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"encoding/pem"

	configv1 "github.com/openshift/api/config/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"

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

func (i *Installer) ensureSecret(ctx context.Context, secrets coreclient.SecretInterface, certificateName string) error {
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

	_, err = secrets.Create(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: certificateName,
		},
		Data: map[string][]byte{
			v1.TLSCertKey:       cb,
			v1.TLSPrivateKeyKey: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
		},
		Type: v1.SecretTypeTLS,
	})
	if errors.IsAlreadyExists(err) {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			s, err := secrets.Get(certificateName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			s.Data = map[string][]byte{
				v1.TLSCertKey:       cb,
				v1.TLSPrivateKeyKey: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
			}
			s.Type = v1.SecretTypeTLS

			_, err = secrets.Update(s)
			return err
		})
	}
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

	err = i.ensureSecret(ctx, i.kubernetescli.CoreV1().Secrets("openshift-config"), i.doc.ID+"-apiserver")
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		apiserver, err := i.configcli.ConfigV1().APIServers().Get("cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		apiserver.Spec.ServingCerts.NamedCertificates = []configv1.APIServerNamedServingCert{
			{
				Names: []string{
					"api." + managedDomain,
				},
				ServingCertificate: configv1.SecretNameReference{
					Name: i.doc.ID + "-apiserver",
				},
			},
		}

		_, err = i.configcli.ConfigV1().APIServers().Update(apiserver)
		return err
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

	err = i.ensureSecret(ctx, i.kubernetescli.CoreV1().Secrets("openshift-ingress"), i.doc.ID+"-ingress")
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ic, err := i.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get("default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		ic.Spec.DefaultCertificate = &v1.LocalObjectReference{
			Name: i.doc.ID + "-ingress",
		}

		_, err = i.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Update(ic)
		return err
	})
}
