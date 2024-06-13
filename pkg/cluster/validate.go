package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/validate"
)

func (m *manager) validateResources(ctx context.Context) error {
	ocDynamicValidator := validate.NewOpenShiftClusterDynamicValidator(
		m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.fpAuthorizer, m.platformWorkloadIdentityRolesByVersion,
	)
	err := ocDynamicValidator.Dynamic(ctx)
	if err != nil {
		return err
	}
	return err
}
