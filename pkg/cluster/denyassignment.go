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

	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		for operatorName, identity := range m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			if identity.ObjectID == "" {
				return fmt.Errorf("createOrUpdateDenyAssignment failed: ObjectID for identity %s is empty", operatorName)
			}
		}
	} else {
		if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile == nil {
			return fmt.Errorf("createOrUpdateDenyAssignment failed: ServicePrincipalProfile is empty")
		}

		if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID == "" {
			return fmt.Errorf("createOrUpdateDenyAssignment failed: SPObjectID is empty")
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
