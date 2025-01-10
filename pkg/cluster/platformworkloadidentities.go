package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
)

func (m *manager) platformWorkloadIdentityIDs(ctx context.Context) error {
	var err error
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return fmt.Errorf("platformWorkloadIdentityIDs called for CSP cluster")
	}

	identities := m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities
	updatedIdentities, err := platformworkloadidentity.GetPlatformWorkloadIdentityIDs(ctx, identities, m.userAssignedIdentities)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = updatedIdentities
		return nil
	})

	return err
}
