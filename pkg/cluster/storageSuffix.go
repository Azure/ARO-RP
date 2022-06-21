package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

func openShiftClusterDocumentMutatorFn(doc *api.OpenShiftClusterDocument) error {
	const ImageRegistry = "imageregistry"

	if doc.OpenShiftCluster.Properties.StorageSuffix == "" {
		var err error

		doc.OpenShiftCluster.Properties.StorageSuffix, err = randomLowerCaseAlphanumericStringWithNoVowels(5)
		if err != nil {
			return err
		}

		doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = ImageRegistry + doc.OpenShiftCluster.Properties.StorageSuffix
	}

	doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = ImageRegistry + doc.OpenShiftCluster.Properties.StorageSuffix

	return nil
}

func (m *manager) ensureStorageSuffix(ctx context.Context) error {
	var err error

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, openShiftClusterDocumentMutatorFn)
	return err
}
