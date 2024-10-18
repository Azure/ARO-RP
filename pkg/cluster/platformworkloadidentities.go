package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) platformWorkloadIdentityIDs(ctx context.Context) error {
	var err error
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return fmt.Errorf("platformWorkloadIdentityIDs called for CSP cluster")
	}

	identities := m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities
	updatedIdentities := make(map[string]api.PlatformWorkloadIdentity, len(identities))

	for i, identity := range identities {
		resourceId, err := arm.ParseResourceID(identity.ResourceID)
		if err != nil {
			return fmt.Errorf("platform workload identity '%s' invalid: %w", i, err)
		}

		identityDetails, err := m.userAssignedIdentities.Get(ctx, resourceId.ResourceGroupName, resourceId.Name, &armmsi.UserAssignedIdentitiesClientGetOptions{})
		if err != nil {
			return fmt.Errorf("error occured when retrieving platform workload identity '%s' details: %w", i, err)
		}

		updatedIdentities[i] = api.PlatformWorkloadIdentity{
			ResourceID:   identity.ResourceID,
			ClientID:     *identityDetails.Properties.ClientID,
			ObjectID:     *identityDetails.Properties.PrincipalID,
		}
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = updatedIdentities
		return nil
	})

	return err
}
