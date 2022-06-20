package cluster

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
)

func (m *manager) ensureStorageSuffix(ctx context.Context) error {
	var err error
	var f database.OpenShiftDocumentMutator = func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.StorageSuffix == "" {
			doc.OpenShiftCluster.Properties.StorageSuffix, err = randomLowerCaseAlphanumericStringWithNoVowels(5)
			if err != nil {
				return err
			}

			doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = "imageregistry" + doc.OpenShiftCluster.Properties.StorageSuffix
		}

		doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = "imageregistry" + doc.OpenShiftCluster.Properties.StorageSuffix

		return nil
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, f)
	return err
}
