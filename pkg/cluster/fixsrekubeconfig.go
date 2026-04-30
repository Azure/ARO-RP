package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) fixSREKubeconfig(ctx context.Context) error {
	if len(m.doc.OpenShiftCluster.Properties.AROSREKubeconfig) > 0 {
		return nil
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	var pg graph.PersistedGraph
	if err := arm.Retryable(ctx, func() error {
		var e error
		pg, e = m.graph.LoadPersisted(ctx, resourceGroup, account)
		return e
	}, m.log, "loading persisted graph"); err != nil {
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
