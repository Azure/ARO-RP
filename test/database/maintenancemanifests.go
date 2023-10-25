package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func injectMaintenanceManifests(c *cosmosdb.FakeMaintenanceManifestDocumentClient, now func() time.Time) {
	c.SetQueryHandler(database.MaintenanceManifestDequeueQuery, func(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceManifestDocumentRawIterator {
		return fakeMaintenanceManifestsDequeue(client, query, options, now)
	})

	c.SetTriggerHandler("renewLease", func(ctx context.Context, doc *api.MaintenanceManifestDocument) error {
		return fakeMaintenanceManifestsRenewLeaseTrigger(ctx, doc, now)
	})
}

func fakeMaintenanceManifestsDequeue(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options, now func() time.Time) cosmosdb.MaintenanceManifestDocumentRawIterator {
	startingIndex, err := fakeMaintenanceManifestsGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeMaintenanceManifestDocumentErroringRawIterator(err)
	}

	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	clusterID := query.Parameters[0].Value

	fmt.Print(clusterID, startingIndex)

	var results []*api.MaintenanceManifestDocument
	for _, r := range input.MaintenanceManifestDocuments {
		if r.ClusterID == clusterID &&
			r.MaintenanceManifest.State == api.MaintenanceManifestStatePending &&
			r.MaintenanceManifest.RunAfter < int(now().Unix()) {
			results = append(results, r)
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].MaintenanceManifest.RunAfter > results[j].MaintenanceManifest.RunAfter && results[i].MaintenanceManifest.Priority > results[j].MaintenanceManifest.Priority
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
