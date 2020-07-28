package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/operator/deploy"
)

func (i *Installer) ensureAROOperator(ctx context.Context) error {
	dep, err := deploy.New(i.log, i.env, i.doc.OpenShiftCluster, i.kubernetescli, i.extcli)
	if err != nil {
		return err
	}
	return dep.CreateOrUpdate()
}

func (i *Installer) aroDeploymentReady() (bool, error) {
	dep, err := deploy.New(i.log, i.env, i.doc.OpenShiftCluster, i.kubernetescli, i.extcli)
	if err != nil {
		return false, err
	}
	return dep.IsReady()
}
