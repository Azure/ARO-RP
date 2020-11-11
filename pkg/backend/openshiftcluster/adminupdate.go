package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

func (m *manager) AdminUpdate(ctx context.Context) error {
	// m.ocDynamicValidator.Dynamic is not called so that it doesn't block an
	// admin update

	i, err := cluster.NewManager(ctx, m.log, m.env, m.db, m.cipher, m.billing, m.doc, m.subscriptionDoc)
	if err != nil {
		return err
	}

	return i.AdminUpgrade(ctx)
}
