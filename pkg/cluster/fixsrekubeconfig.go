package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) fixSREKubeconfig(ctx context.Context) error {
	if len(m.doc.OpenShiftCluster.Properties.AROSREKubeconfig) > 0 {
		return nil
	}

	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, account)
	if err != nil {
		return err
	}

	aroSREInternalClient, err := m.generateAROSREKubeconfig(pg)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.AROSREKubeconfig = aroSREInternalClient
		return nil
	})
	return err
}
