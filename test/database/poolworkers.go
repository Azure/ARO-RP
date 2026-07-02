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

func fakePoolWorkerRenewLeaseTrigger(_ context.Context, doc *api.PoolWorkerDocument, now func() time.Time) error {
	doc.LeaseExpires = int(now().Unix()) + 60
	return nil
}

func fakePoolWorkerGetMasterQuery(client cosmosdb.PoolWorkerDocumentClient, q *cosmosdb.Query, opts *cosmosdb.Options, now func() time.Time) cosmosdb.PoolWorkerDocumentRawIterator {
	input, err := client.ListAll(context.Background(), opts)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	out := []*api.PoolWorkerDocument{}
	for _, r := range input.PoolWorkerDocuments {
		if r.ID != q.Parameters[0].Value {
			continue
		}
		if string(r.WorkerType) != q.Parameters[0].Value {
			continue
		}
		if time.Unix(int64(r.LeaseExpires), 0).After(now()) {
			continue
		}
		out = append(out, r)
	}

	return cosmosdb.NewFakePoolWorkerDocumentIterator(out, 0)
}

func fakePoolWorkerGetAllButMasterHandler(client cosmosdb.PoolWorkerDocumentClient, q *cosmosdb.Query, opts *cosmosdb.Options, now func() time.Time) cosmosdb.PoolWorkerDocumentRawIterator {
	input, err := client.ListAll(context.Background(), opts)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}
	if input == nil {
		return cosmosdb.NewFakePoolWorkerDocumentIterator(nil, 0)
	}

	out := []*api.PoolWorkerDocument{}
	for _, r := range input.PoolWorkerDocuments {
		if r.ID == q.Parameters[0].Value {
			continue
		}
		if string(r.WorkerType) != q.Parameters[0].Value {
			continue
		}
		// XXX: This does not test for TTL -- we need to add saving a Timestamp to gocosmosdb
		out = append(out, r)
	}
	return cosmosdb.NewFakePoolWorkerDocumentIterator(out, 0)
}

func injectPoolWorkers(c *cosmosdb.FakePoolWorkerDocumentClient, now func() time.Time) {
	c.SetQueryHandler(database.PoolWorkerGetMasterQuery, func(client cosmosdb.PoolWorkerDocumentClient, query *cosmosdb.Query, opts *cosmosdb.Options) cosmosdb.PoolWorkerDocumentRawIterator {
		return fakePoolWorkerGetMasterQuery(client, query, opts, now)
	})
	c.SetQueryHandler(database.PoolWorkerGetWorkersQuery, func(client cosmosdb.PoolWorkerDocumentClient, query *cosmosdb.Query, opts *cosmosdb.Options) cosmosdb.PoolWorkerDocumentRawIterator {
		return fakePoolWorkerGetAllButMasterHandler(client, query, opts, now)
	})
	c.SetTriggerHandler("renewLease", func(ctx context.Context, doc *api.PoolWorkerDocument) error {
		return fakePoolWorkerRenewLeaseTrigger(ctx, doc, now)
	})
	c.SetSorter(func(in []*api.PoolWorkerDocument) {
		slices.SortFunc(in, func(a, b *api.PoolWorkerDocument) int { return CompareIDable(a, b) })
	})
}
