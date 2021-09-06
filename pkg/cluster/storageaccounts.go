package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) populateRegistryStorageAccountName(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName != "" {
		return nil
	}

	rc, err := m.registryclient.ImageregistryV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = rc.Spec.Storage.Azure.AccountName
		return nil
	})
	return err
}
