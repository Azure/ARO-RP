package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func injectMaintenanceManifests(c *cosmosdb.FakeMaintenanceManifestDocumentClient, now func() time.Time) {
	c.SetQueryHandler(database.MaintenanceManifestQueryForCluster, func(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceManifestDocumentRawIterator {
		return fakeMaintenanceManifestsForCluster(client, query, options, now)
	})
	c.SetQueryHandler(database.MaintenanceManifestDequeueQueryForCluster, func(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceManifestDocumentRawIterator {
		return fakeMaintenanceManifestsDequeueForCluster(client, query, options, now)
	})

	c.SetQueryHandler(database.MaintenanceManifestQueueOverallQuery, func(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceManifestDocumentRawIterator {
		return fakeMaintenanceManifestsQueuedAll(client, query, options, now)
	})

	c.SetTriggerHandler("renewLease", func(ctx context.Context, doc *api.MaintenanceManifestDocument) error {
		return fakeMaintenanceManifestsRenewLeaseTrigger(ctx, doc, now)
	})
}

func fakeMaintenanceManifestsDequeueForCluster(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options, now func() time.Time) cosmosdb.MaintenanceManifestDocumentRawIterator {
	startingIndex, err := fakeMaintenanceManifestsGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeMaintenanceManifestDocumentErroringRawIterator(err)
	}

	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	clusterResourceID := query.Parameters[0].Value

	var results []*api.MaintenanceManifestDocument
	for _, r := range input.MaintenanceManifestDocuments {
		if r.ClusterResourceID != clusterResourceID {
			continue
		}
		if r.MaintenanceManifest.State != api.MaintenanceManifestStatePending {
			continue
		}
		if r.LeaseExpires > 0 && int64(r.LeaseExpires) < time.Now().Unix() {
			continue
		}
		if int64(r.MaintenanceManifest.RunAfter) < time.Now().Unix() {
			continue
		}
		results = append(results, r)
	}

	return cosmosdb.NewFakeMaintenanceManifestDocumentIterator(results, startingIndex)
}

func fakeMaintenanceManifestsForCluster(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options, now func() time.Time) cosmosdb.MaintenanceManifestDocumentRawIterator {
	startingIndex, err := fakeMaintenanceManifestsGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeMaintenanceManifestDocumentErroringRawIterator(err)
	}

	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	clusterResourceID := query.Parameters[0].Value

	var results []*api.MaintenanceManifestDocument
	for _, r := range input.MaintenanceManifestDocuments {
		if r.ClusterResourceID != clusterResourceID {
			continue
		}
		results = append(results, r)
	}

	slices.SortFunc(results, func(a, b *api.MaintenanceManifestDocument) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return cosmosdb.NewFakeMaintenanceManifestDocumentIterator(results, startingIndex)
}

func fakeMaintenanceManifestsQueuedAll(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options, now func() time.Time) cosmosdb.MaintenanceManifestDocumentRawIterator {
	startingIndex, err := fakeMaintenanceManifestsGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeMaintenanceManifestDocumentErroringRawIterator(err)
	}

	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	var results []*api.MaintenanceManifestDocument
	for _, r := range input.MaintenanceManifestDocuments {
		if r.MaintenanceManifest.State != api.MaintenanceManifestStatePending {
			continue
		}
		if r.LeaseExpires > 0 && int64(r.LeaseExpires) < time.Now().Unix() {
			continue
		}

		results = append(results, r)
	}

	slices.SortFunc(results, func(a, b *api.MaintenanceManifestDocument) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return cosmosdb.NewFakeMaintenanceManifestDocumentIterator(results, startingIndex)
}

func fakeMaintenanceManifestsRenewLeaseTrigger(ctx context.Context, doc *api.MaintenanceManifestDocument, now func() time.Time) error {
	doc.LeaseExpires = int(now().Unix()) + 60
	return nil
}

func fakeMaintenanceManifestsGetContinuation(options *cosmosdb.Options) (startingIndex int, err error) {
	if options != nil && options.Continuation != "" {
		startingIndex, err = strconv.Atoi(options.Continuation)
	}
	return
}
