package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type mockQueryManifestClient struct {
	cosmosdb.MaintenanceManifestDocumentClient
	gotPartitionKey string
	gotQuery        *cosmosdb.Query
	gotOptions      *cosmosdb.Options
}

func (m *mockQueryManifestClient) Query(partitionKey string, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceManifestDocumentRawIterator {
	m.gotPartitionKey = partitionKey
	m.gotQuery = query
	m.gotOptions = options
	return nil
}

func TestGetByClusterResourceIDQueryWiring(t *testing.T) {
	ctx := context.Background()
	const clusterResourceID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster"

	for _, tt := range []struct {
		name       string
		opts       MaintenanceManifestListOptions
		wantQuery  string
		wantParams []cosmosdb.Parameter
	}{
		{
			name: "after only",
			opts: MaintenanceManifestListOptions{After: 100},
			wantQuery: MaintenanceManifestQueryForClusterAfter,
			wantParams: []cosmosdb.Parameter{
				{Name: "@clusterResourceID", Value: clusterResourceID},
				{Name: "@after", Value: "100"},
			},
		},
		{
			name: "after and before",
			opts: MaintenanceManifestListOptions{After: 100, Before: new(int64(200))},
			wantQuery: MaintenanceManifestQueryForClusterAfterBefore,
			wantParams: []cosmosdb.Parameter{
				{Name: "@clusterResourceID", Value: clusterResourceID},
				{Name: "@after", Value: "100"},
				{Name: "@before", Value: "200"},
			},
		},
		{
			name: "after zero for full history",
			opts: MaintenanceManifestListOptions{After: 0},
			wantQuery: MaintenanceManifestQueryForClusterAfter,
			wantParams: []cosmosdb.Parameter{
				{Name: "@clusterResourceID", Value: clusterResourceID},
				{Name: "@after", Value: "0"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockQueryManifestClient{}
			db := NewMaintenanceManifestsWithProvidedClient(mockClient, nil, "test-uuid", uuid.DefaultGenerator)

			_, err := db.GetByClusterResourceID(ctx, clusterResourceID, "", tt.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mockClient.gotPartitionKey != clusterResourceID {
				t.Errorf("partition key: expected %q, got %q", clusterResourceID, mockClient.gotPartitionKey)
			}

			if mockClient.gotQuery.Query != tt.wantQuery {
				t.Errorf("query: expected %q, got %q", tt.wantQuery, mockClient.gotQuery.Query)
			}

			if len(mockClient.gotQuery.Parameters) != len(tt.wantParams) {
				t.Fatalf("expected %d query parameters, got %d", len(tt.wantParams), len(mockClient.gotQuery.Parameters))
			}
			for i, want := range tt.wantParams {
				got := mockClient.gotQuery.Parameters[i]
				if got.Name != want.Name || got.Value != want.Value {
					t.Errorf("parameter[%d]: expected {%s %v}, got {%s %v}", i, want.Name, want.Value, got.Name, got.Value)
				}
			}
		})
	}
}
