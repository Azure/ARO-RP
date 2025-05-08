package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

func (m *manager) ensureBillingRecord(ctx context.Context) error {
	return m.billing.Ensure(ctx, m.doc, m.subscriptionDoc)
}
