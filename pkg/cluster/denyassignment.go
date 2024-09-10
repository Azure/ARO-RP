package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) createOrUpdateDenyAssignment(ctx context.Context) error {
	if m.env.FeatureIsSet(env.FeatureDisableDenyAssignments) {
		return nil
	}

	// needed for AdminUpdate so it would not block other steps
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		for _, i := range m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			if i.ObjectID == "" {
				m.log.Print(fmt.Sprintf("skipping createOrUpdateDenyAssignment: ObjectID for identity %s is empty", i))
				return nil
			}
		}
	} else {
		if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile == nil {
			m.log.Print("skipping createOrUpdateDenyAssignment: ServicePrincipalProfile is empty")
			return nil
		}

		if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID == "" {
			m.log.Print("skipping createOrUpdateDenyAssignment: SPObjectID is empty")
			return nil
		}
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			m.denyAssignment(),
		},
	}

	return arm.DeployTemplate(ctx, m.log, m.deployments, resourceGroup, "storage", t, nil)
}
