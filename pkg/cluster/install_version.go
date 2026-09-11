package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) openShiftVersionFromVersion(ctx context.Context) (*api.OpenShiftVersion, error) {
	// Use the enhanced version resolver with subscription data for AFEC flag support
	return m.openShiftClusterDocumentVersioner.GetWithSubscription(ctx, m.doc, m.dbOpenShiftVersions, m.env, m.installViaHive, m.subscriptionDoc)
}
