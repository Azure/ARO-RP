package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"slices"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func fakeMonitoringRenewLeaseTrigger(_ context.Context, doc *api.MonitorDocument, now func() time.Time) error {
	doc.LeaseExpires = int(now().Unix()) + 60
	return nil
}

func fakeMonitorGetMasterQuery(client cosmosdb.MonitorDocumentClient, _ *cosmosdb.Query, opts *cosmosdb.Options, now func() time.Time) cosmosdb.MonitorDocumentRawIterator {
	input, err := client.ListAll(context.Background(), opts)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	out := []*api.MonitorDocument{}
	for _, r := range input.MonitorDocuments {
		if r.ID != "master" {
			continue
		}
		if time.Unix(int64(r.LeaseExpires), 0).After(now()) {
			continue
		}
		out = append(out, r)
	}

	return cosmosdb.NewFakeMonitorDocumentIterator(out, 0)
}

func fakeMonitorGetAllButMasterHandler(client cosmosdb.MonitorDocumentClient, _ *cosmosdb.Query, opts *cosmosdb.Options, now func() time.Time) cosmosdb.MonitorDocumentRawIterator {
	input, err := client.ListAll(context.Background(), opts)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}
	if input == nil {
		return cosmosdb.NewFakeMonitorDocumentIterator(nil, 0)
	}

	out := []*api.MonitorDocument{}
	for _, r := range input.MonitorDocuments {
		if r.ID == "master" {
			continue
		}
		// XXX: This does not test for TTL -- we need to add saving a Timestamp to gocosmosdb
		out = append(out, r)
	}
	return cosmosdb.NewFakeMonitorDocumentIterator(out, 0)
}

func injectMonitors(c *cosmosdb.FakeMonitorDocumentClient, now func() time.Time) {
	c.SetQueryHandler(database.MonitorsTryLeaseQuery, func(client cosmosdb.MonitorDocumentClient, query *cosmosdb.Query, opts *cosmosdb.Options) cosmosdb.MonitorDocumentRawIterator {
		return fakeMonitorGetMasterQuery(client, query, opts, now)
	})
	c.SetQueryHandler(database.MonitorsWorkerQuery, func(client cosmosdb.MonitorDocumentClient, query *cosmosdb.Query, opts *cosmosdb.Options) cosmosdb.MonitorDocumentRawIterator {
		return fakeMonitorGetAllButMasterHandler(client, query, opts, now)
	})
	c.SetTriggerHandler("renewLease", func(ctx context.Context, doc *api.MonitorDocument) error {
		return fakeMonitoringRenewLeaseTrigger(ctx, doc, now)
	})
	c.SetSorter(func(in []*api.MonitorDocument) {
		slices.SortFunc(in, func(a, b *api.MonitorDocument) int { return CompareIDable(a, b) })
	})
}
