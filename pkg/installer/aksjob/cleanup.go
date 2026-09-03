package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// Cleanup removes installer resources (namespace and all contained resources)
func (m *manager) Cleanup(ctx context.Context, namespace string) error {
	m.log.Infof("cleaning up namespace %s", namespace)

	// Check if namespace exists
	_, err := m.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			m.log.Info("namespace already deleted")
			return nil
		}
		m.log.Warnf("failed to check namespace: %v", err)
		return nil // Non-fatal
	}

	// Delete namespace with grace period
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: pointerutils.ToPtr(int64(30)),
	}

	err = m.client.CoreV1().Namespaces().Delete(ctx, namespace, deleteOptions)
	if err != nil {
		if errors.IsNotFound(err) {
			m.log.Info("namespace already deleted")
			return nil
		}
		m.log.Warnf("failed to delete namespace: %v", err)
		return nil // Non-fatal cleanup failures
	}

	m.log.Infof("namespace %s deleted", namespace)
	return nil
}
