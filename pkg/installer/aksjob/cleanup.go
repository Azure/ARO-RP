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
//
// The namespace is shared by every install in the region, so the Job is only
// removed once it is confirmed to carry executionID. A caller resuming from a
// stale InstallerProfile therefore cannot delete a Job belonging to a later
// execution, nor the Secrets named after it.
func (m *manager) Cleanup(ctx context.Context, namespace, jobName, executionID string) error {
	m.log.Infof("cleaning up installer execution %s/%s", namespace, jobName)

	job, err := m.client.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		// Already gone; fall through and remove any Secrets left behind.
	case err != nil:
		return fmt.Errorf("failed to get installer Job %s/%s: %w", namespace, jobName, err)
	case job.Labels["execution-id"] != executionID:
		return fmt.Errorf("refusing to clean up installer Job %s/%s: belongs to execution %q, not %q",
			namespace, jobName, job.Labels["execution-id"], executionID)
	default:
		propagation := metav1.DeletePropagationBackground
		err = m.client.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{
			PropagationPolicy: &propagation,
		})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete installer Job %s/%s: %w", namespace, jobName, err)
		}
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
