package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// migrateStorageAccounts redeploys storage accounts with firewall rules preventing external access
// The encryption flag is set to false/disabled for legacy storage accounts.
func (m *manager) migrateStorageAccounts(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	ocpSubnets, err := m.subnetsWithServiceEndpoint(ctx, storageServiceEndpoint)
	if err != nil {
		return err
	}

	clusterStorageAccountName := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix
	registryStorageAccountName := m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName

	for _, storageAccountName := range []string{clusterStorageAccountName, registryStorageAccountName} {
		parameters := mgmtstorage.AccountUpdateParameters{
			AccountPropertiesUpdateParameters: m.storageAccountProperties(storageAccountName, ocpSubnets),
		}
		if _, err := m.storage.UpdateAccount(ctx, resourceGroup, storageAccountName, parameters); err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) populateRegistryStorageAccountName(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName != "" {
		return nil
	}

	rc, err := m.imageregistrycli.ImageregistryV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if rc.Spec.Storage.Azure == nil {
			return fmt.Errorf("azure storage field is nil in image registry config")
		}

		doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = rc.Spec.Storage.Azure.AccountName
		return nil
	})
	return err
}
