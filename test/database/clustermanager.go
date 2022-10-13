package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

type SortableClusterManagerConfigurationDocument []*api.ClusterManagerConfigurationDocument

func (a SortableClusterManagerConfigurationDocument) Len() int      { return len(a) }
func (a SortableClusterManagerConfigurationDocument) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortableClusterManagerConfigurationDocument) Less(i, j int) bool {
	return strings.Compare(a[i].Key, a[j].Key) < 0
}

func injectClusterManager(c *cosmosdb.FakeClusterManagerConfigurationDocumentClient) {
	c.SetQueryHandler(database.ClusterManagerConfigurationsGetQuery, fakeClusterManagerConfigurationsGetQuery)

	c.SetSorter(func(in []*api.ClusterManagerConfigurationDocument) {
		sort.Sort(SortableClusterManagerConfigurationDocument(in))
	})
}

func fakeClusterManagerConfigurationsGetQuery(client cosmosdb.ClusterManagerConfigurationDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.ClusterManagerConfigurationDocumentRawIterator {
	docs, err := fakeClusterManagerGetAllDocuments(client)
	if err != nil {
		return cosmosdb.NewFakeClusterManagerConfigurationDocumentErroringRawIterator(err)
	}
	return cosmosdb.NewFakeClusterManagerConfigurationDocumentIterator(docs, 0)
}

func fakeClusterManagerGetAllDocuments(client cosmosdb.ClusterManagerConfigurationDocumentClient) ([]*api.ClusterManagerConfigurationDocument, error) {
	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	docs := input.ClusterManagerConfigurationDocuments
	sort.Sort(SortableClusterManagerConfigurationDocument(docs))
	return docs, nil
}
