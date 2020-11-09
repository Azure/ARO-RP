package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

func (m *manager) Update(ctx context.Context) error {
	// TODO: m.ocDynamicValidator.Dynamic is not called because it should run on
	// an enriched oc.  Neither are we enriching oc here currently, nor does
	// Dynamic() support running on an enriched oc.

	return nil
}
