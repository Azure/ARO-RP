package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/install"
)

func (m *Manager) Update(ctx context.Context) error {
	// TODO: m.ocDynamicValidator.Dynamic is not called because it should run on
	// an enriched oc.  Neither are we enriching oc here currently, nor does
	// Dynamic() support running on an enriched oc.

	i, err := install.NewInstaller(ctx, m.log, m.env, m.db, m.billing, m.doc, m.subscriptionDoc)
	if err != nil {
		return err
	}

	return i.Update(ctx)
}
