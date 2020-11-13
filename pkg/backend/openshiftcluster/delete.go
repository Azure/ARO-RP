package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/cluster"
)

func (m *manager) Delete(ctx context.Context) error {
	i, err := cluster.NewManager(ctx, m.log, m.env, m.db, m.cipher, m.billing, m.doc, m.subscriptionDoc)
	if err != nil {
		return err
	}

	return i.Delete(ctx)
}
