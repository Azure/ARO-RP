package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
)

func managedDomain(tc mimo.TaskContext) (string, error) {
	env := tc.Environment()
	clusterProperties := tc.GetOpenShiftClusterProperties()

	managedDomain, err := dns.ManagedDomain(env, clusterProperties.ClusterProfile.Domain)
	if err != nil {
		// if it fails the belt&braces check then not much we can do
		return "", err
	}

	if managedDomain == "" {
		return "", nil
	}
	return managedDomain, nil
}

func RotateAPIServerCertificate(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	managedDomain, err := managedDomain(th)
	if err != nil {
		return mimo.TerminalError(err)
	}
	if managedDomain == "" {
		th.SetResultMessage("apiserver certificate is not managed")
		return nil
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}

	env := th.Environment()
	secretName := th.GetClusterUUID() + "-apiserver"
	kv := env.ClusterKeyvault()

	isCustom, err := isCustomAPIServerCertificate(ctx, kv, secretName, managedDomain)
	if err != nil {
		return mimo.TransientError(err)
	}
	if isCustom {
		th.SetResultMessage("apiserver certificate is custom; skipping rotation")
		return nil
	}

	for _, namespace := range []string{"openshift-config", "openshift-azure-operator"} {
		err = cluster.EnsureTLSSecretFromKeyvault(
			ctx, kv, ch, types.NamespacedName{Namespace: namespace, Name: secretName}, secretName,
		)
		if err != nil {
			return mimo.TransientError(err)
		}
	}

	return nil
}

func isCustomAPIServerCertificate(ctx context.Context, kv azsecrets.Client, secretName, managedDomain string) (bool, error) {
	if managedDomain == "" {
		return false, nil
	}

	bundle, err := kv.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return false, err
	}
	if bundle.Value == nil {
		return false, fmt.Errorf("secret %s has no value", secretName)
	}

	cert, err := utilpem.ParseFirstCertificate([]byte(*bundle.Value))
	if err != nil {
		return false, err
	}

	expectedDNS := "api." + managedDomain
	for _, dnsName := range cert.DNSNames {
		if strings.EqualFold(dnsName, expectedDNS) {
			return false, nil
		}
	}

	return true, nil
}

func EnsureAPIServerServingCertificateConfiguration(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}

	managedDomain, err := managedDomain(th)
	if err != nil {
		return mimo.TerminalError(err)
	}
	if managedDomain == "" {
		th.SetResultMessage("apiserver certificate is not managed")
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		apiserver := &configv1.APIServer{}

		err := ch.GetOne(ctx, types.NamespacedName{Name: "cluster"}, apiserver)
		if err != nil {
			if kerrors.IsNotFound(err) {
				// apiserver not being found is probably unrecoverable
				return mimo.TerminalError(err)
			}
			return mimo.TransientError(err)
		}

		apiserver.Spec.ServingCerts.NamedCertificates = []configv1.APIServerNamedServingCert{
			{
				Names: []string{
					"api." + managedDomain,
				},
				ServingCertificate: configv1.SecretNameReference{
					Name: th.GetClusterUUID() + "-apiserver",
				},
			},
		}

		err = ch.Update(ctx, apiserver)
		if err != nil {
			if kerrors.IsConflict(err) {
				return err
			} else {
				return mimo.TransientError(err)
			}
		}
		return nil
	})
}
