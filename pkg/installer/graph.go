package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) persistGraph(ctx context.Context, g graph.Graph) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	clusterStorageAccountName := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	exists, err := m.graph.Exists(ctx, resourceGroup, clusterStorageAccountName)
	if err != nil || exists {
		return err
	}

	// the graph is quite big, so we store it in a storage account instead of in cosmosdb
	return m.graph.Save(ctx, resourceGroup, clusterStorageAccountName, g)
}
