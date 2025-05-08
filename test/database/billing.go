package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func fakeBillingCreationTimestampTrigger(ctx context.Context, doc *api.BillingDocument) error {
	doc.Billing.CreationTime = int(time.Now().Unix())
	return nil
}

func fakeBillingDeletionTimestampTrigger(ctx context.Context, doc *api.BillingDocument) error {
	doc.Billing.DeletionTime = int(time.Now().Unix())
	return nil
}

func injectBilling(c *cosmosdb.FakeBillingDocumentClient) {
	c.SetTriggerHandler("setCreationBillingTimeStamp", fakeBillingCreationTimestampTrigger)
	c.SetTriggerHandler("setDeletionBillingTimeStamp", fakeBillingDeletionTimestampTrigger)
}
