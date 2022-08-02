package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) hiveCreateNamespace(ctx context.Context) error {
	m.log.Info("creating a namespace in the hive cluster")
	if m.hiveClusterManager == nil {
		// TODO(hive): remove this once we have Hive everywhere
		m.log.Info("skipping: no hive cluster manager")
		return nil
	}

	if m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace != "" {
		m.log.Info("skipping: namespace already exists")
		return nil
	}

	namespace, err := m.hiveClusterManager.CreateNamespace(ctx)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.HiveProfile.Namespace = namespace.Name
		return nil
	})

	return err
}

func (m *manager) hiveEnsureResources(ctx context.Context) error {
	m.log.Info("registering with hive")
	if m.hiveClusterManager == nil {
		// TODO(hive): remove this once we have Hive everywhere
		m.log.Info("skipping: no hive cluster manager")
		return nil
	}

	return m.hiveClusterManager.CreateOrUpdate(ctx, m.subscriptionDoc, m.doc)
}

func (m *manager) hiveDeleteResources(ctx context.Context) error {
	m.log.Info("deregistering cluster with hive")
	if m.hiveClusterManager == nil {
		// TODO(hive): remove this once we have Hive everywhere
		m.log.Info("skipping: no hive cluster manager")
		return nil
	}

	namespace := m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace
	if namespace == "" {
		m.log.Info("skipping: no hive namespace in cluster document")
		return nil
	}

	return m.hiveClusterManager.Delete(ctx, namespace)
}
