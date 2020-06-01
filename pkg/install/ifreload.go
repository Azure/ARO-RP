package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/ifreload"
)

func (i *Installer) ensureIfReload(ctx context.Context) error {
	ir := ifreload.New(i.log, i.env, i.kubernetescli, i.securitycli)
	return ir.CreateOrUpdate(ctx)
}
