package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (i *Installer) createBillingRecord(ctx context.Context) error {
	_, err := i.billing.Create(ctx, &api.BillingDocument{
		ID:                        i.doc.ID,
		Key:                       i.doc.Key,
		ClusterResourceGroupIDKey: i.doc.ClusterResourceGroupIDKey,
		Billing: &api.Billing{
			TenantID: i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
			Location: i.doc.OpenShiftCluster.Location,
		},
	})

	if err != nil {
		return err
	}
	return nil
}
