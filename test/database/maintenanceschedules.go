package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"slices"
	"strconv"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func injectMaintenanceSchedules(c *cosmosdb.FakeMaintenanceScheduleDocumentClient) {
	c.SetQueryHandler(database.MaintenanceScheduleQueryValid, func(client cosmosdb.MaintenanceScheduleDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceScheduleDocumentRawIterator {
		return fakeMaintenanceSchedulesAllValid(client, query, options)
	})

	c.SetSorter(func(in []*api.MaintenanceScheduleDocument) {
		slices.SortFunc(in, func(a, b *api.MaintenanceScheduleDocument) int { return CompareIDable(a, b) })
	})
}

func fakeMaintenanceSchedulesAllValid(client cosmosdb.MaintenanceScheduleDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceScheduleDocumentRawIterator {
	startingIndex, err := fakeMaintenanceSchedulesGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeMaintenanceScheduleDocumentErroringRawIterator(err)
	}

	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	var results []*api.MaintenanceScheduleDocument
	for _, r := range input.MaintenanceScheduleDocuments {
		if r.MaintenanceSchedule.State != api.MaintenanceScheduleStateEnabled {
			continue
		}
		results = append(results, r)
	}

	slices.SortFunc(results, func(a, b *api.MaintenanceScheduleDocument) int { return CompareIDable(a, b) })

	return cosmosdb.NewFakeMaintenanceScheduleDocumentIterator(results, startingIndex)
}

func fakeMaintenanceSchedulesGetContinuation(options *cosmosdb.Options) (startingIndex int, err error) {
	if options != nil && options.Continuation != "" {
		startingIndex, err = strconv.Atoi(options.Continuation)
	}
	return
}
