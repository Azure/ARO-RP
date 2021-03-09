package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) createOrUpdateDenyAssignment(ctx context.Context) error {
	if m.env.DeploymentMode() != deployment.Production {
		// only need this upgrade in production, where there are DenyAssignments
		return nil
	}

	// needed for AdminUpdate so it would not block other steps
	if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID == "" {
		m.log.Print("skipping createOrUpdateDenyAssignment: SPObjectID is empty")
		return nil
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			m.denyAssignment(),
			m.clusterServicePrincipalRoleDefinition(),
		},
	}

	return m.deployARMTemplate(ctx, resourceGroup, "storage", t, nil)
}
