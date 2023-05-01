package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/hive"
)

func (m *manager) runInstallerInAKS(ctx context.Context) error {
	aksmanager, err := hive.NewAKSManagerFromHiveManager(m.hiveClusterManager)
	if err != nil {
		return err
	}

	version, err := m.openShiftVersionFromVersion(ctx)
	if err != nil {
		return err
	}

	return aksmanager.Install(ctx, m.subscriptionDoc, m.doc, version)
}
