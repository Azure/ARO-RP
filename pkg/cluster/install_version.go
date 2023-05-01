package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
)

func (m *manager) openShiftVersionFromVersion(ctx context.Context) (*api.OpenShiftVersion, error) {
	return m.openShiftClusterDocumentVersioner.Get(ctx, m.doc, m.dbOpenShiftVersions, m.env, m.installStrategy == liveconfig.HiveStrategy)
}
