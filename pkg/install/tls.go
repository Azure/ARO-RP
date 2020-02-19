package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
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
		err = i.keyvault.CreateCertificate(ctx, c.certificateName, c.commonName)
		if err != nil {
			return err
		}
	}

	for _, c := range certs {
		i.log.Printf("waiting for certificate %s", c.certificateName)
		err = i.keyvault.WaitForCertificateOperation(ctx, c.certificateName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) ensureSecret(ctx context.Context, secrets coreclient.SecretInterface, certificateName string) error {
	key, certs, err := i.keyvault.GetSecret(ctx, certificateName)
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

	restConfig, err := restconfig.RestConfig(ctx, i.env, i.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	err = i.ensureSecret(ctx, cli.CoreV1().Secrets("openshift-config"), i.doc.ID+"-apiserver")
	if err != nil {
		return err
	}

	ccli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		apiserver, err := ccli.ConfigV1().APIServers().Get("cluster", metav1.GetOptions{})
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

		_, err = ccli.ConfigV1().APIServers().Update(apiserver)
		return err
	})
	if err != nil {
		return err
	}

	ocli, err := operatorclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.log.Print("waiting for apiservers")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		apiserver, err := ocli.OperatorV1().KubeAPIServers().Get("cluster", metav1.GetOptions{})
		if err == nil {
			m := make(map[string]operatorv1.ConditionStatus, len(apiserver.Status.Conditions))
			for _, cond := range apiserver.Status.Conditions {
				m[cond.Type] = cond.Status
			}
			if m["Available"] == operatorv1.ConditionTrue && m["Progressing"] == operatorv1.ConditionFalse {
				return true, nil
			}
		}
		return false, nil
	}, timeoutCtx.Done())
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

	restConfig, err := restconfig.RestConfig(ctx, i.env, i.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	err = i.ensureSecret(ctx, cli.CoreV1().Secrets("openshift-ingress"), i.doc.ID+"-ingress")
	if err != nil {
		return err
	}

	ocli, err := operatorclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ic, err := ocli.OperatorV1().IngressControllers("openshift-ingress-operator").Get("default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		ic.Spec.DefaultCertificate = &v1.LocalObjectReference{
			Name: i.doc.ID + "-ingress",
		}

		_, err = ocli.OperatorV1().IngressControllers("openshift-ingress-operator").Update(ic)
		return err
	})
	if err != nil {
		return err
	}

	i.log.Print("waiting for ingress controller")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		ic, err := ocli.OperatorV1().IngressControllers("openshift-ingress-operator").Get("default", metav1.GetOptions{})
		if err == nil && ic.Status.ObservedGeneration == ic.Generation {
			for _, cond := range ic.Status.Conditions {
				if cond.Type == operatorv1.OperatorStatusTypeAvailable && cond.Status == operatorv1.ConditionTrue {
					return true, nil
				}
			}
		}
		return false, nil
	}, timeoutCtx.Done())
}
