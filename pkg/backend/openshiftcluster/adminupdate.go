package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/cluster"
)

func (m *Manager) AdminUpdate(ctx context.Context) error {
	// m.ocDynamicValidator.Dynamic is not called so that it doesn't block an
	// admin update

	i, err := cluster.New(ctx, m.log, m.env, m.fp, m.gl, m.dialer, m.fakearm, m.version, m.db, m.cipher, m.billing, m.doc, m.subscriptionDoc, m.clustersKeyvaultURI)
	if err != nil {
		return err
	}

	return i.AdminUpgrade(ctx)
}
