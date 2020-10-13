package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/routefix"
)

func (m *manager) ensureRouteFix(ctx context.Context) error {
	rf := routefix.New(m.log, m.env, m.kubernetescli, m.securitycli)
	return rf.CreateOrUpdate(ctx)
}
