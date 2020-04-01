package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

func (m *Manager) Update(ctx context.Context) error {
	return m.ocDynamicValidator.Dynamic(ctx, m.doc.OpenShiftCluster)
}
