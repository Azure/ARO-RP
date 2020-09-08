package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/routefix"
)

func (i *manager) ensureRouteFix(ctx context.Context) error {
	rf := routefix.New(i.log, i.env, i.version, i.kubernetescli, i.securitycli)
	return rf.CreateOrUpdate(ctx)
}
