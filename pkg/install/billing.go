package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func (i *Installer) createBillingRecord(ctx context.Context) error {
	return cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		_, err = i.billing.Create(ctx, &api.BillingDocument{
			ID:                        i.doc.ID,
			Key:                       i.doc.Key,
			ClusterResourceGroupIDKey: i.doc.ClusterResourceGroupIDKey,
			Billing: &api.Billing{
				TenantID:        i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
				Location:        i.doc.OpenShiftCluster.Location,
				CreationTime:    -1,
				LastBillingTime: -1,
			},
		})
		// If create return a conflict, this means row is already present in database, updating timestamps
		if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
			_, err := i.billing.Patch(ctx, i.doc.ID, func(billingdoc *api.BillingDocument) (bool, error) {
				return false, nil
			})
			if err != nil {
				return err
			}
			return nil
		}
		if err != nil {
			return err
		}
		return nil
	})
}
