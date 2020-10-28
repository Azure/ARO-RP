package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/operator/deploy"
)

func (m *manager) ensureAROOperator(ctx context.Context) error {
	dep, err := deploy.New(m.log, m.env, m.doc.OpenShiftCluster, m.kubernetescli, m.extcli, m.arocli)
	if err != nil {
		return err
	}
	return dep.CreateOrUpdate(ctx)
}

func (m *manager) aroDeploymentReady(ctx context.Context) (bool, error) {
	dep, err := deploy.New(m.log, m.env, m.doc.OpenShiftCluster, m.kubernetescli, m.extcli, m.arocli)
	if err != nil {
		return false, err
	}
	return dep.IsReady(ctx)
}
