package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cleanup removes resources owned by one execution while retaining the shared
// namespace and ServiceAccount provisioned with the SVC cluster.
func (m *manager) Cleanup(ctx context.Context, namespace, jobName string) error {
	m.log.Infof("cleaning up installer execution %s/%s", namespace, jobName)

	propagation := metav1.DeletePropagationBackground
	err := m.client.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete installer Job %s/%s: %w", namespace, jobName, err)
	}

	names := resourceNames(jobName)
	for _, name := range []string{
		names.inputsSecret,
		names.boundKeySecret,
		names.manifestsSecret,
		names.pullSecret,
	} {
		err = m.client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete installer Secret %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}
