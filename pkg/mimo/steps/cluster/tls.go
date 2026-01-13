package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/hashicorp/go-multierror"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func ingressCertName(th mimo.TaskContext) string {
	return th.GetClusterUUID() + "-ingress"
}

func apiserverCertName(th mimo.TaskContext) string {
	return th.GetClusterUUID() + "-apiserver"
}

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

func signedCertificatesDisabled(tc mimo.TaskContext) bool {
	if tc.Environment().FeatureIsSet(env.FeatureDisableSignedCertificates) {
		tc.SetResultMessage("signed certificates disabled")
		return true
	}
	return false
}

func RotateManagedCertificates(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	if signedCertificatesDisabled(th) {
		return nil
	}

	taskEnv := th.Environment()
	managedDomain, err := managedDomain(th)
	if err != nil {
		return mimo.TerminalError(err)
	}
	if managedDomain == "" {
		th.SetResultMessage("cluster certificates are not managed")
		return nil
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}

	var errs error

	// Attempt both certificates -- one failing should not stop the other, but
	// it will still return a TransientError
	for _, cert := range []struct {
		secretName      string
		targetNamespace string
	}{
		{secretName: apiserverCertName(th), targetNamespace: "openshift-config"},
		{secretName: ingressCertName(th), targetNamespace: "openshift-ingress"},
	} {
		secrets, err := cluster.TLSSecretsFromKeyVault(
			ctx, taskEnv.ClusterKeyvault(), []types.NamespacedName{
				{Namespace: cert.targetNamespace, Name: cert.secretName},
				{Namespace: "openshift-azure-operator", Name: cert.secretName},
			}, cert.secretName,
		)
		if err != nil {
			th.Log().Errorf("failed to load %s from keyvault: %s", cert.secretName, err)
			errs = multierror.Append(errs, err)
			continue
		}

		err = ch.Ensure(ctx, secrets...)
		if err != nil {
			th.Log().Errorf("failed to save %s: %s", cert.secretName, err)
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return mimo.TransientError(errs)
	}

	return nil
}

func EnsureAPIServerServingCertificateConfiguration(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	if signedCertificatesDisabled(th) {
		return nil
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
					Name: apiserverCertName(th),
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

func EnsureIngressServingCertificateConfiguration(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	if signedCertificatesDisabled(th) {
		return nil
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
		th.SetResultMessage("cluster certificate is not managed")
		return nil
	}

	outerErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ic := &operatorv1.IngressController{}

		err := ch.GetOne(ctx, types.NamespacedName{Namespace: "openshift-ingress-operator", Name: "default"}, ic)
		ic.Spec.DefaultCertificate = &corev1.LocalObjectReference{
			Name: ingressCertName(th),
		}
		if err != nil {
			return err
		}

		return ch.Update(ctx, ic)
	})

	if outerErr != nil {
		if kerrors.IsNotFound(outerErr) {
			// apiserver not being found is probably unrecoverable
			return mimo.TerminalError(outerErr)
		} else {
			return mimo.TransientError(outerErr)
		}
	}

	return nil
}
