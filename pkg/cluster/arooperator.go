package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/operator/deploy"
)

func (m *manager) ensureAROOperator(ctx context.Context) error {
	dep, err := deploy.New(m.log, m.env, m.doc.OpenShiftCluster, m.arocli, m.extensionscli, m.kubernetescli)
	if err != nil {
		m.log.Errorf("cannot ensureAROOperator.New: %s", err.Error())
		return err
	}
	err = dep.CreateOrUpdate(ctx)
	if err != nil {
		m.log.Errorf("cannot ensureAROOperator.CreateOrUpdate: %s", err.Error())
	}
	return err
}

func (m *manager) aroDeploymentReady(ctx context.Context) (bool, error) {
	dep, err := deploy.New(m.log, m.env, m.doc.OpenShiftCluster, m.arocli, m.extensionscli, m.kubernetescli)
	if err != nil {
		return false, err
	}
	return dep.IsReady(ctx)
}
