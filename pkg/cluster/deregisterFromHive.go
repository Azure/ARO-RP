package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
)

func (m *manager) deregisterClusterFromHive(ctx context.Context) error {
	if m.hiveClusterManager == nil {
		return errors.New("no hive cluster manager, skipping")
	}
	namespace := m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace

	if namespace == "" {
		m.log.Info("no hive namespace name in cluster document, skipping hive deregistration")
		return nil
	}

	return m.hiveClusterManager.Delete(ctx, namespace)
}
