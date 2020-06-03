package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/operator/deploy"
	"github.com/Azure/ARO-RP/pkg/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

func (i *Installer) readyToDeployAroOperator() (bool, error) {
	restConfig, err := restconfig.RestConfig(i.env, i.doc.OpenShiftCluster)
	if err != nil {
		return false, err
	}
	dh, err := dynamichelper.New(i.log, restConfig, dynamichelper.UpdatePolicy{})
	if err != nil {
		return false, nil
	}
	_, err = dh.Get(context.TODO(), "SecurityContextConstraints", "", "privileged")
	return err == nil, nil
}

func (i *Installer) ensureAroOperator(ctx context.Context) error {
	i.log.Print("Installing ARO operator resources")

	dep, err := deploy.New(i.log, i.env, i.doc.OpenShiftCluster, i.kubernetescli, i.securitycli, i.arocli)
	if err != nil {
		i.log.Warnf("deploy.New %v", err)
		return err
	}
	return dep.CreateOrUpdate(ctx)
}

func (i *Installer) aroDeploymentReady() (bool, error) {
	dep, err := deploy.New(i.log, i.env, i.doc.OpenShiftCluster, i.kubernetescli, i.securitycli, i.arocli)
	if err != nil {
		i.log.Warnf("deploy.New %v", err)
		return false, err
	}
	return dep.IsReady()
}
