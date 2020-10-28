package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

func (m *manager) fixPullSecret(ctx context.Context) error {
	// TODO: this function does not currently reapply a pull secret in
	// development mode.

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ps, err := m.kubernetescli.CoreV1().Secrets("openshift-config").Get(ctx, "pull-secret", metav1.GetOptions{})
		if err != nil {
			return err
		}

		pullSecret, changed, err := pullsecret.SetRegistryProfiles(string(ps.Data[v1.DockerConfigJsonKey]), m.doc.OpenShiftCluster.Properties.RegistryProfiles...)
		if err != nil {
			return err
		}

		if !changed {
			return nil
		}

		ps.Data[v1.DockerConfigJsonKey] = []byte(pullSecret)

		_, err = m.kubernetescli.CoreV1().Secrets("openshift-config").Update(ctx, ps, metav1.UpdateOptions{})
		return err
	})
}
