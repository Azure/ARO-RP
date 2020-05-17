package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/operator/deploy"
)

func (i *Installer) ensureAroOperator(ctx context.Context) error {
	i.log.Print("Installing ARO operator resources")
	dep, err := deploy.New(i.log, i.env, i.doc.OpenShiftCluster, i.kubernetescli, i.securitycli, i.arocli)
	if err != nil {
		return err
	}
	return dep.CreateOrUpdate(ctx)
}
