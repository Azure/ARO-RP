package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) hiveCreateNamespace(ctx context.Context) error {
	m.log.Info("creating a namespace in the hive cluster")

	if m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace != "" {
		m.log.Info("skipping: namespace already exists")
		return nil
	}

	namespace, err := m.hiveClusterManager.CreateNamespace(ctx, m.doc.ID)
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
	return m.hiveClusterManager.CreateOrUpdate(ctx, m.subscriptionDoc, m.doc)
}

func (m *manager) hiveClusterDeploymentReady(ctx context.Context) (deploymentReady bool, retry bool, err error) {
	m.log.Info("waiting for cluster deployment to become ready")
	deploymentReady, err = m.hiveClusterManager.IsClusterDeploymentReady(ctx, m.doc)
	if err != nil {
		return deploymentReady, false, err
	} else {
		if !deploymentReady {
			return deploymentReady, true, err
		} else {
			return deploymentReady, false, err
		}
	}
}

func (m *manager) hiveClusterInstallationComplete(ctx context.Context) (installationComplete bool, retry bool, err error) {
	m.log.Info("waiting for cluster installation to complete")
	installationComplete, err = m.hiveClusterManager.IsClusterDeploymentReady(ctx, m.doc)
	if err != nil {
		return installationComplete, false, err
	} else {
		if !installationComplete {
			return installationComplete, true, err
		} else {
			return installationComplete, false, err
		}
	}
}

func (m *manager) hiveResetCorrelationData(ctx context.Context) error {
	m.log.Info("resetting correlation data for hive")
	return m.hiveClusterManager.ResetCorrelationData(ctx, m.doc)
}

func (m *manager) hiveDeleteResources(ctx context.Context) error {
	m.log.Info("deregistering cluster with hive")
	namespace := m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace
	if namespace == "" {
		m.log.Info("skipping: no hive namespace in cluster document")
		return nil
	}

	return m.hiveClusterManager.Delete(ctx, m.doc)
}
