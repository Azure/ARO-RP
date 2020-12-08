package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
)

func (m *manager) validateResources(ctx context.Context) error {
	ocDynamicValidator := validate.NewOpenShiftClusterDynamicValidator(
		m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.fpAuthorizer,
	)
	return ocDynamicValidator.Dynamic(ctx)
}

func (m *manager) validateQuota(ctx context.Context) error {
	// we don't validate the quota after we have potentially deployed our VMs,
	// since the quota validator does not take resources we have deployed into
	// account
	if m.doc.OpenShiftCluster.Properties.Install == nil || m.doc.OpenShiftCluster.Properties.Install.Phase != api.InstallPhaseBootstrap {
		return nil
	}

	qv, err := validate.NewAzureQuotaValidator(ctx, m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc)
	if err != nil {
		return err
	}

	return qv.Validate(ctx)
}
