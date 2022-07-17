package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/openshift/installer/pkg/asset/installconfig/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/hive"
)

func (m *manager) hiveCreateNamespace(ctx context.Context) error {
	m.log.Info("creating a namespace in the hive cluster")
	if m.hiveClusterManager == nil {
		// TODO(hive): remove this if once we have Hive everywhere
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
		// TODO(hive): remove this if once we have Hive everywhere
		m.log.Info("skipping: no hive cluster manager")
		return nil
	}

	m.log.Info("collecting registration data from the new cluster")
	parameters, err := collectDataForHive(m.subscriptionDoc, m.doc)
	if err != nil {
		return err
	}

	err = m.hiveClusterManager.CreateOrUpdate(ctx, parameters)
	if err != nil {
		return err
	}

	return err
}

func (m *manager) hiveDeleteResources(ctx context.Context) error {
	m.log.Info("deregistering cluster with hive")
	if m.hiveClusterManager == nil {
		// TODO(hive): remove this if once we have Hive everywhere
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

// TODO(hive): Consider moving somewhere like pkg/hive/util.go
func collectDataForHive(subscriptionDoc *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) (*hive.CreateOrUpdateParameters, error) {
	// TODO(hive): When hive support first party principles we'll need to send both first party and cluster service principles
	clusterSP := azure.Credentials{
		TenantID:       subscriptionDoc.Subscription.Properties.TenantID,
		SubscriptionID: subscriptionDoc.ID,
		ClientID:       doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID,
		ClientSecret:   string(doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret),
	}

	clusterSPBytes, err := json.Marshal(clusterSP)
	if err != nil {
		return nil, err
	}

	return &hive.CreateOrUpdateParameters{
		Namespace:        doc.OpenShiftCluster.Properties.HiveProfile.Namespace,
		ClusterName:      doc.OpenShiftCluster.Name,
		Location:         doc.OpenShiftCluster.Location,
		InfraID:          doc.OpenShiftCluster.Properties.InfraID,
		ClusterID:        doc.ID,
		KubeConfig:       string(doc.OpenShiftCluster.Properties.AROServiceKubeconfig),
		ServicePrincipal: string(clusterSPBytes),
	}, nil
}
