package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

// populateCreatedAt updates DB document with cluster creation date from the default namespace
// This is a temporary piece of code to populate the relevant field for older clusters.
// TODO(mikalai): Remove the function after a round of admin updates
func (m *manager) populateCreatedAt(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.Properties.CreatedAt.IsZero() {
		return nil
	}

	ns, err := m.kubernetescli.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.CreatedAt = ns.CreationTimestamp.Time
		return nil
	})
	return err
}
