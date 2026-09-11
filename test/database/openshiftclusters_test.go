package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestFakeGetIDsByClusterResourceGroupID(t *testing.T) {
	ctx := context.Background()

	const (
		subscriptionID = "00000000-0000-0000-0000-000000000000"
		matchingRG     = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/managed-rg"
		otherRG        = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/other-rg"
	)

	db, _ := NewFakeOpenShiftClusters()

	seed := []*api.OpenShiftClusterDocument{
		{ID: "doc-1", Key: strings.ToLower(GetResourcePath(subscriptionID, "cluster1")), ClusterResourceGroupIDKey: matchingRG, OpenShiftCluster: &api.OpenShiftCluster{ID: "id-1"}},
		{ID: "doc-2", Key: strings.ToLower(GetResourcePath(subscriptionID, "cluster2")), ClusterResourceGroupIDKey: otherRG, OpenShiftCluster: &api.OpenShiftCluster{ID: "id-2"}},
	}
	for _, doc := range seed {
		if _, err := db.Create(ctx, doc); err != nil {
			t.Fatal(err)
		}
	}

	result, err := db.GetIDsByClusterResourceGroupID(ctx, subscriptionID, matchingRG)
	if err != nil {
		t.Fatal(err)
	}

	if result.Count != 1 {
		t.Fatalf("count: expected 1, got %d", result.Count)
	}

	got := result.OpenShiftClusterDocuments[0]
	if got.ID != "doc-1" {
		t.Errorf("id: expected %q, got %q", "doc-1", got.ID)
	}
	// The id-only projection must not carry the full payload.
	if got.OpenShiftCluster != nil {
		t.Errorf("expected OpenShiftCluster nil, got %+v", got.OpenShiftCluster)
	}
}
