package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/operator/deploy"
)

func (i *manager) ensureAROOperator(ctx context.Context) error {
	dep, err := deploy.New(i.log, i.env, i.version, i.gl, i.dialer, i.doc.OpenShiftCluster, i.kubernetescli, i.extcli, i.arocli)
	if err != nil {
		return err
	}
	return dep.CreateOrUpdate()
}

func (i *manager) aroDeploymentReady(ctx context.Context) (bool, error) {
	dep, err := deploy.New(i.log, i.env, i.version, i.gl, i.dialer, i.doc.OpenShiftCluster, i.kubernetescli, i.extcli, i.arocli)
	if err != nil {
		return false, err
	}
	return dep.IsReady()
}
