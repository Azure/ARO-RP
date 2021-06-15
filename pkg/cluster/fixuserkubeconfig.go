package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// fixUserAdminKubeconfig adds shorter kubeconfig for user to return
// TODO(mjudeikis): This will one 1 year kubeconfig. We should add check for -90 days
// and rotate it.
func (m *manager) fixUserAdminKubeconfig(ctx context.Context) error {
	if len(m.doc.OpenShiftCluster.Properties.UserAdminKubeconfig) > 0 {
		return nil
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return err
	}

	aroUserClient, err := m.generateUserAdminKubeconfig(pg)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.UserAdminKubeconfig = aroUserClient.File.Data
		return nil
	})
	return err
}
