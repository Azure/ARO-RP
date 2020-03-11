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
	_, err := i.billing.Create(ctx, &api.BillingDocument{
		ID:                        i.doc.ID,
		Key:                       i.doc.Key,
		ClusterResourceGroupIDKey: i.doc.ClusterResourceGroupIDKey,
		Billing: &api.Billing{
			TenantID: i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
			Location: i.doc.OpenShiftCluster.Location,
		},
	})
	// If create return a conflict, this means row is already present in database
	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		return nil
	}

	return err
}
