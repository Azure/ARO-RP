package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) createOrUpdateDenyAssignment(ctx context.Context) error {
	if m.env.FeatureIsSet(env.FeatureDisableDenyAssignments) {
		return nil
	}

	var validationErr error
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		for operatorName, identity := range m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			if identity.ObjectID == "" {
				validationErr = fmt.Errorf("ObjectID for identity %s is empty", operatorName)
				break
			}
		}
	} else {
		if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile == nil {
			validationErr = fmt.Errorf("ServicePrincipalProfile is empty")
		} else if m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID == "" {
			validationErr = fmt.Errorf("SPObjectID is empty")
		}
	}

	if validationErr != nil {
		if m.doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateAdminUpdating {
			m.log.Printf("skipping createOrUpdateDenyAssignment: %v", validationErr)
			return nil
		}
		return fmt.Errorf("createOrUpdateDenyAssignment failed: %w", validationErr)
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
