package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/install"
)

func (m *Manager) AdminUpdate(ctx context.Context) error {
	// m.ocDynamicValidator.Dynamic is not called

	i, err := install.NewInstaller(ctx, m.log, m.env, m.db, m.billing, m.doc)
	if err != nil {
		return err
	}

	return i.AdminUpgrade(ctx)
}
