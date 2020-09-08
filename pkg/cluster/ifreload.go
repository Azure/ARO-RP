package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/ifreload"
)

func (i *manager) ensureIfReload(ctx context.Context) error {
	ir := ifreload.New(i.log, i.env, i.version, i.kubernetescli, i.securitycli)
	return ir.CreateOrUpdate(ctx)
}
