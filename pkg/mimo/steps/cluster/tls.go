package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func RotateAPIServerCertificate(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}

	env := th.Environment()
	secretName := th.GetClusterUUID() + "-apiserver"

	for _, namespace := range []string{"openshift-config", "openshift-azure-operator"} {
		err = cluster.EnsureTLSSecretFromKeyvault(
			ctx, env.ClusterKeyvault(), ch, types.NamespacedName{Namespace: namespace, Name: secretName}, secretName,
		)
		if err != nil {
			return err
		}
	}

	return nil
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

	env := th.Environment()
	clusterProperties := th.GetOpenShiftClusterProperties()

	managedDomain, err := dns.ManagedDomain(env, clusterProperties.ClusterProfile.Domain)
	if err != nil {
		// if it fails the belt&braces check then not much we can do
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
