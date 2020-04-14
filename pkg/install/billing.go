package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

func (i *Installer) createBillingRecord(ctx context.Context) error {
	return i.billing.Create(ctx, i.doc)
}
