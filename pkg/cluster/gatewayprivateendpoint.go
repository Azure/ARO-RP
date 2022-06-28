package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// ensureGatewayUpgrade checks to see if the cluster should have the gateway
// enabled but doesn't yet.  If so, it sets the master subnet policies, deploys
// the private endpoint, approves the gateway PE/PLS connection, creates the
// gateway database record and updates the model with the private endpoint IP.
func (m *manager) ensureGatewayUpgrade(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled ||
		m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateEndpointIP != "" {
		return nil
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	err := m.setMasterSubnetPolicies(ctx)
	if err != nil {
		m.log.Print(err)
		return nil
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      []*arm.Resource{m.networkPrivateEndpoint()},
	}
	err = arm.DeployTemplate(ctx, m.log, m.deployments, resourceGroup, "gatewayprivateendpoint", t, nil)
	if err != nil {
		m.log.Print(err)
		return nil
	}

	err = m.ensureGatewayCreate(ctx)
	if err != nil {
		m.log.Print(err)
		return nil
	}

	return nil
}
