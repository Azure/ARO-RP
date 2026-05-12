package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
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

	// The error should be wrapped, so we need to unwrap it
	var cosmosErr *cosmosdb.Error
	if !errors.As(err, &cosmosErr) {
		t.Fatalf("expected wrapped *cosmosdb.Error, got %T", err)
	}

	if cosmosErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status code 500, got %d", cosmosErr.StatusCode)
	}

	// Verify the error message contains descriptive text
	if !strings.Contains(err.Error(), "creating OpenShift cluster") {
		t.Errorf("expected error message to contain 'creating OpenShift cluster', got %q", err.Error())
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
	expectedMsg := fmt.Sprintf("OpenShiftClusters Replace returned nil document with nil error for key %q", doc.Key)
	if cosmosErr.Message != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, cosmosErr.Message)
	}
}
