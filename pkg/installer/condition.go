package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// condition functions should return an error only if it's not retryable
// if a condition function encounters a retryable error it should return false, nil.

func (m *manager) bootstrapConfigMapReady(ctx context.Context) (bool, error) {
	cm, err := m.kubernetescli.CoreV1().ConfigMaps("kube-system").Get(ctx, "bootstrap", metav1.GetOptions{})
	if err != nil && m.env.IsLocalDevelopmentMode() {
		m.log.Printf("bootstrapConfigMapReady condition error %s", err)
	}
	return err == nil && cm.Data["status"] == "complete", nil
}
