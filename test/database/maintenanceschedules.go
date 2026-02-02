package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func injectMaintenanceSchedules(c *cosmosdb.FakeMaintenanceScheduleDocumentClient, now func() time.Time) {
	c.SetTriggerHandler("renewLease", func(ctx context.Context, doc *api.MaintenanceScheduleDocument) error {
		return fakeMaintenanceSchedulesRenewLeaseTrigger(ctx, doc, now)
	})
}

func fakeMaintenanceSchedulesRenewLeaseTrigger(ctx context.Context, doc *api.MaintenanceScheduleDocument, now func() time.Time) error {
	doc.LeaseExpires = int(now().Unix()) + 60
	return nil
}
