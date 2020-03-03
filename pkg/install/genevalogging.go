package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/genevalogging"
)

func (i *Installer) ensureGenevaLogging(ctx context.Context) error {
	gl := genevalogging.New(i.log, i.env, i.doc.OpenShiftCluster, i.kubernetescli, i.securitycli)
	return gl.CreateOrUpdate(ctx)
}
