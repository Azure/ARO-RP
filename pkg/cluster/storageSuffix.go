package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

func setDocStorageSuffix(doc *api.OpenShiftClusterDocument) error {
	isDocStorageSuffixValid := doc.OpenShiftCluster.Properties.StorageSuffix != ""
	if isDocStorageSuffixValid {
		return nil
	}

	storageSuffix, err := randomLowerCaseAlphanumericStringWithNoVowels(10)
	doc.OpenShiftCluster.Properties.StorageSuffix = storageSuffix

	return err
}

func mutateStorageSuffix(doc *api.OpenShiftClusterDocument) error {
	err := setDocStorageSuffix(doc)
	if err != nil {
		return err
	}

	doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = "imageregistry" + doc.OpenShiftCluster.Properties.StorageSuffix

	return nil
}

func (m *manager) ensureStorageSuffix(ctx context.Context) error {
	updatedDoc, err := m.db.PatchWithLease(ctx, m.doc.Key, mutateStorageSuffix)
	m.doc = updatedDoc

	return err
}
