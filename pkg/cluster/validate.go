package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/ARO-RP/pkg/validate"
)

func (m *manager) validateResources(ctx context.Context) error {
	var clusterMSICredential azcore.TokenCredential
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		clusterMSICredential = m.userAssignedIdentities.GetClusterMSICredential()
	}
	return validate.NewOpenShiftClusterDynamicValidator(
		m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.fpAuthorizer, m.armRoleDefinitions, m.clusterMsiFederatedIdentityCredentials, m.userAssignedIdentities, m.platformWorkloadIdentityRolesByVersion, clusterMSICredential,
	).Dynamic(ctx)
}
