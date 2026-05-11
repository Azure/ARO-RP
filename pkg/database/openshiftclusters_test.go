package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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
		Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster",
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

	expectedErrMsg := "cosmosdb create returned nil document with nil error"
	if err.Error() != expectedErrMsg {
		t.Errorf("expected error message %q, got %q", expectedErrMsg, err.Error())
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

	expectedErrMsg := "cosmosdb replace returned nil document with nil error"
	if err.Error() != expectedErrMsg {
		t.Errorf("expected error message %q, got %q", expectedErrMsg, err.Error())
	}
}
