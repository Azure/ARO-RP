package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

// mockNilReturningClient is a minimal mock that returns (nil, nil) from Create and Replace
// to test the regression where CosmosDB could return (nil, nil) and we didn't handle it
type mockNilReturningClient struct {
	cosmosdb.OpenShiftClusterDocumentClient
	returnNilNil bool
}

func (m *mockNilReturningClient) Create(ctx context.Context, partitionKey string, doc *api.OpenShiftClusterDocument, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
	if m.returnNilNil {
		return nil, nil
	}
	return doc, nil
}

func (m *mockNilReturningClient) Replace(ctx context.Context, partitionKey string, doc *api.OpenShiftClusterDocument, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
	if m.returnNilNil {
		return nil, nil
	}
	return doc, nil
}

// TestCreateReturnsErrorWhenCosmosDBReturnsNilNil verifies that when the CosmosDB client
// returns (nil, nil) from Create, the wrapper returns a proper error instead of (nil, nil).
// This is a regression test for a bug where CosmosDB could return (nil, nil) and we would
// propagate it to callers, leading to nil pointer dereferences.
func TestCreateReturnsErrorWhenCosmosDBReturnsNilNil(t *testing.T) {
	ctx := context.Background()

	mockClient := &mockNilReturningClient{returnNilNil: true}
	db := NewOpenShiftClustersWithProvidedClient(mockClient, nil, "test-uuid", uuid.DefaultGenerator)

	doc := &api.OpenShiftClusterDocument{
		Key:          "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster",
		PartitionKey: "00000000-0000-0000-0000-000000000000",
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: "test-id",
		},
	}

	result, err := db.Create(ctx, doc)

	// Verify we get an error instead of (nil, nil)
	if err == nil {
		t.Fatal("expected error when CosmosDB returns (nil, nil), got nil")
	}

	if result != nil {
		t.Fatalf("expected nil result when error occurs, got %v", result)
	}

	cosmosErr, ok := err.(*cosmosdb.Error)
	if !ok {
		t.Fatalf("expected *cosmosdb.Error, got %T", err)
	}

	if cosmosErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status code 500, got %d", cosmosErr.StatusCode)
	}

	// Verify the error message contains descriptive text
	expectedMsg := fmt.Sprintf("creating OpenShift cluster: CosmosDB returned nil document with no error for key %q and partition key %q", doc.Key, doc.PartitionKey)
	if cosmosErr.Message != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, cosmosErr.Message)
	}
}

// TestUpdateReturnsErrorWhenCosmosDBReturnsNilNil verifies that when the CosmosDB client
// returns (nil, nil) from Replace, the wrapper returns a proper error instead of (nil, nil).
// This is a regression test for a bug where CosmosDB could return (nil, nil) and we would
// propagate it to callers, leading to nil pointer dereferences.
func TestUpdateReturnsErrorWhenCosmosDBReturnsNilNil(t *testing.T) {
	ctx := context.Background()

	mockClient := &mockNilReturningClient{returnNilNil: true}
	db := NewOpenShiftClustersWithProvidedClient(mockClient, nil, "test-uuid", uuid.DefaultGenerator)

	doc := &api.OpenShiftClusterDocument{
		Key:          "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster",
		PartitionKey: "00000000-0000-0000-0000-000000000000",
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: "test-id",
		},
	}

	result, err := db.Update(ctx, doc)

	// Verify we get an error instead of (nil, nil)
	if err == nil {
		t.Fatal("expected error when CosmosDB returns (nil, nil), got nil")
	}

	if result != nil {
		t.Fatalf("expected nil result when error occurs, got %v", result)
	}

	cosmosErr, ok := err.(*cosmosdb.Error)
	if !ok {
		t.Fatalf("expected *cosmosdb.Error, got %T", err)
	}

	if cosmosErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status code 500, got %d", cosmosErr.StatusCode)
	}

	// Verify the error message contains descriptive text
	expectedMsg := fmt.Sprintf("OpenShiftClusters Replace returned nil document with nil error for key %q and partition key %q", doc.Key, doc.PartitionKey)
	if cosmosErr.Message != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, cosmosErr.Message)
	}
}

// mockQueryAllClient captures the query and partition key passed to QueryAll and
// returns a canned result, so tests can assert the projection query wiring without
// a live CosmosDB.
type mockQueryAllClient struct {
	cosmosdb.OpenShiftClusterDocumentClient
	gotPartitionKey string
	gotQuery        *cosmosdb.Query
	result          *api.OpenShiftClusterDocuments
}

func (m *mockQueryAllClient) QueryAll(ctx context.Context, partitionKey string, query *cosmosdb.Query, options *cosmosdb.Options) (*api.OpenShiftClusterDocuments, error) {
	m.gotPartitionKey = partitionKey
	m.gotQuery = query
	return m.result, nil
}

// TestGetIDsByClusterResourceGroupIDProjectsIDOnly verifies that
// GetIDsByClusterResourceGroupID wires up the id-only projection query with the
// correct partition key and @resourceGroupID parameter, and that it returns the
// documents unchanged (ID populated, OpenShiftCluster nil).
func TestGetIDsByClusterResourceGroupIDProjectsIDOnly(t *testing.T) {
	ctx := context.Background()

	const (
		partitionKey    = "00000000-0000-0000-0000-000000000000"
		resourceGroupID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/managed-rg"
	)

	mockClient := &mockQueryAllClient{
		result: &api.OpenShiftClusterDocuments{
			Count: 1,
			OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
				{ID: "doc-id-1"},
			},
		},
	}
	db := NewOpenShiftClustersWithProvidedClient(mockClient, nil, "test-uuid", uuid.DefaultGenerator)

	docs, err := db.GetIDsByClusterResourceGroupID(ctx, partitionKey, resourceGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockClient.gotPartitionKey != partitionKey {
		t.Errorf("partition key: expected %q, got %q", partitionKey, mockClient.gotPartitionKey)
	}

	if mockClient.gotQuery.Query != OpenshiftClustersResourceGroupIDOnlyQuery {
		t.Errorf("query: expected %q, got %q", OpenshiftClustersResourceGroupIDOnlyQuery, mockClient.gotQuery.Query)
	}

	// Guard the projection intent itself: doc.id must be the only projected field,
	// so it must be immediately followed by FROM (rejecting e.g. "SELECT doc.id, doc.key").
	if q := mockClient.gotQuery.Query; !strings.HasPrefix(q, "SELECT doc.id FROM ") {
		t.Errorf("query is not an id-only projection: %q", q)
	}

	if len(mockClient.gotQuery.Parameters) != 1 {
		t.Fatalf("expected 1 query parameter, got %d", len(mockClient.gotQuery.Parameters))
	}
	if param := mockClient.gotQuery.Parameters[0]; param.Name != "@resourceGroupID" || param.Value != resourceGroupID {
		t.Errorf("parameter: expected {@resourceGroupID %q}, got {%s %v}", resourceGroupID, param.Name, param.Value)
	}

	if docs.Count != 1 {
		t.Errorf("count: expected 1, got %d", docs.Count)
	}
	if len(docs.OpenShiftClusterDocuments) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs.OpenShiftClusterDocuments))
	}
	if got := docs.OpenShiftClusterDocuments[0].ID; got != "doc-id-1" {
		t.Errorf("id: expected %q, got %q", "doc-id-1", got)
	}
	if docs.OpenShiftClusterDocuments[0].OpenShiftCluster != nil {
		t.Error("expected OpenShiftCluster to be nil for an id-only projection")
	}
}
