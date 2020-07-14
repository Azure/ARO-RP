package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/install"
)

func (m *Manager) Create(ctx context.Context) error {
	i, err := install.NewInstaller(ctx, m.log, m.env, m.db, m.billing, m.doc, m.subscriptionDoc)
	if err != nil {
		return err
	}
	return i.Install(ctx)
}
