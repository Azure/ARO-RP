package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/openshift/installer/pkg/asset/installconfig/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) collectDataForHive(ctx context.Context) (*hive.WorkloadCluster, error) {
	m.log.Info("collecting registration data from the new cluster")
	if m.hiveClusterManager == nil {
		return nil, errors.New("no hive cluster manager, skipping")
	}

	// TODO: When hive support first party principles we'll need to send both first party and cluster service principles
	clusterSP := azure.Credentials{
		TenantID:       m.subscriptionDoc.Subscription.Properties.TenantID,
		SubscriptionID: m.subscriptionDoc.ID,
		ClientID:       m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID,
		ClientSecret:   string(m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret),
	}

	clusterSPBytes, err := json.Marshal(clusterSP)
	if err != nil {
		return nil, err
	}

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	hiveWorkloadCluster := &hive.WorkloadCluster{
		SubscriptionID:    m.subscriptionDoc.ID,
		ClusterName:       m.doc.OpenShiftCluster.Name,
		ResourceGroupName: resourceGroupName,
		Location:          m.doc.OpenShiftCluster.Location,
		InfraID:           m.doc.OpenShiftCluster.Properties.InfraID,
		ClusterID:         m.doc.ID,
		KubeConfig:        string(m.doc.OpenShiftCluster.Properties.AROServiceKubeconfig),
		ServicePrincipal:  string(clusterSPBytes),
	}

	return hiveWorkloadCluster, nil
}

func (m *manager) registerWithHive(ctx context.Context, workloadCluster *hive.WorkloadCluster) error {
	m.log.Info("registering with hive")
	if m.hiveClusterManager == nil {
		return errors.New("no hive cluster manager, skipping")
	}

	cd, err := m.hiveClusterManager.Register(ctx, workloadCluster)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.HiveProfile = api.HiveProfile{
			Namespace: cd.Namespace,
		}
		return nil
	})

	return err
}

// verifyRegistration is being run in steps.Condition with fail=false
// as we don't want to fail on Hive errors during the post-installation
// cluster adoption.
// We must return true in case of successful adoption.
// Returning false means that we want to continue waiting for adoption.
// Returning non-nil error means that we give up waiting.
func (m *manager) verifyRegistration(ctx context.Context) (bool, error) {
	m.log.Info("verifying cluster registration in hive")
	if m.hiveClusterManager == nil {
		return false, errors.New("no hive cluster manager, skipping")
	}

	isConnected, reason, err := m.hiveClusterManager.IsConnected(ctx, m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace)
	if err != nil {
		m.log.Infof("error getting hive registration status: %s", err)
		return false, nil
	}

	if !isConnected {
		m.log.Infof("hive is not able to connect to cluster %s", reason)
		return false, nil
	}

	m.log.Info("cluster adopted successfully")
	return true, nil
}
