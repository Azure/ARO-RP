package cluster

import "context"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func (m *manager) fixUUID(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.UUID == "" {
		return m.ensureUUID(ctx)
	}

	return nil
}
