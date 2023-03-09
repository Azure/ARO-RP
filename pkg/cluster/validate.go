package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api/validate"
)

func (m *manager) validateResources(ctx context.Context) error {
	ocDynamicValidator, err := validate.NewOpenShiftClusterDynamicValidator(
		ctx, m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.fpAuthorizer,
	)
	if err != nil {
		return err
	}
	return ocDynamicValidator.Dynamic(ctx)
}
