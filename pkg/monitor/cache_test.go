package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
)

type cacheTestOperation int

const (
	Upsert cacheTestOperation = iota
	Delete
	NoOp
)

func TestUpsertAndDelete(t *testing.T) {
	// Setup single monitor for all test operations
	env := createTestEnvironmentWithLocalCosmos(t)
	defer env.LocalCosmosCleanup()
	testMon := env.CreateTestMonitor("test-cache")
	// Set owned buckets for the entire test sequence
	ownedBuckets := []int{1, 2, 5}
	for _, bucket := range ownedBuckets {
		testMon.buckets[bucket] = struct{}{}
	}

	type operation struct {
		name      string
		action    cacheTestOperation
		clusterID string
		bucket    int
		state     api.ProvisioningState
		validate  func(*testing.T, string, *monitor)
	}

	operations := []operation{
		{
			name:      "upsert new document in owned bucket",
			action:    Upsert,
			clusterID: "cluster-1",
			bucket:    5,
			state:     api.ProvisioningStateSucceeded,
			validate: func(t *testing.T, stepName string, mon *monitor) {
				if len(mon.docs) != 1 {
					t.Errorf("%s: expected 1 document, got %d", stepName, len(mon.docs))
				}
				cacheDoc, exists := mon.docs["cluster-1"]
				if !exists {
					t.Fatalf("%s: document was not added to cache", stepName)
				}
				if cacheDoc.doc.OpenShiftCluster.Properties.ProvisioningState != api.ProvisioningStateSucceeded {
					t.Errorf("%s: expected state Succeeded, got %v", stepName, cacheDoc.doc.OpenShiftCluster.Properties.ProvisioningState)
				}
				if cacheDoc.stop == nil {
					t.Errorf("%s: expected worker to be started for owned bucket", stepName)
				}
			},
		},
		{
			name:      "upsert document in non-owned bucket",
			action:    Upsert,
			clusterID: "cluster-2",
			bucket:    10, // not in owned buckets
			state:     api.ProvisioningStateSucceeded,
			validate: func(t *testing.T, stepName string, mon *monitor) {
				if len(mon.docs) != 2 {
					t.Errorf("%s: expected 2 documents, got %d", stepName, len(mon.docs))
				}
				cacheDoc, exists := mon.docs["cluster-2"]
				if !exists {
					t.Fatalf("%s: document was not added to cache", stepName)
				}
				if cacheDoc.stop != nil {
					t.Errorf("%s: expected no worker for non-owned bucket", stepName)
				}
			},
		},
		{
			name:      "update existing document state",
			action:    Upsert,
			clusterID: "cluster-1",
			bucket:    5,
			state:     api.ProvisioningStateUpdating,
			validate: func(t *testing.T, stepName string, mon *monitor) {
				if len(mon.docs) != 2 {
					t.Errorf("%s: expected 2 documents, got %d", stepName, len(mon.docs))
				}
				cacheDoc, exists := mon.docs["cluster-1"]
				if !exists {
					t.Fatalf("%s: document not found in cache", stepName)
				}
				if cacheDoc.doc.OpenShiftCluster.Properties.ProvisioningState != api.ProvisioningStateUpdating {
					t.Errorf("%s: expected state Updating, got %v", stepName, cacheDoc.doc.OpenShiftCluster.Properties.ProvisioningState)
				}
				if cacheDoc.stop == nil {
					t.Errorf("%s: worker should still exist after update", stepName)
				}
			},
		},
		{
			name:      "add third document",
			action:    Upsert,
			clusterID: "cluster-3",
			bucket:    2,
			state:     api.ProvisioningStateSucceeded,
			validate: func(t *testing.T, stepName string, mon *monitor) {
				if len(mon.docs) != 3 {
					t.Errorf("%s: expected 3 documents, got %d", stepName, len(mon.docs))
				}
				cacheDoc, exists := mon.docs["cluster-3"]
				if !exists {
					t.Fatalf("%s: document was not added to cache", stepName)
				}
				if cacheDoc.stop == nil {
					t.Errorf("%s: expected worker for owned bucket", stepName)
				}
			},
		},
		{
			name:      "delete document with worker",
			action:    Delete,
			clusterID: "cluster-1",
			bucket:    5,
			state:     api.ProvisioningStateDeleting,
			validate: func(t *testing.T, stepName string, mon *monitor) {
				if len(mon.docs) != 2 {
					t.Errorf("%s: expected 2 documents after deletion, got %d", stepName, len(mon.docs))
				}
				if _, exists := mon.docs["cluster-1"]; exists {
					t.Errorf("%s: document should have been deleted from cache", stepName)
				}
				// Verify other documents still exist
				if _, exists := mon.docs["cluster-2"]; !exists {
					t.Errorf("%s: cluster-2 should not be affected by deletion", stepName)
				}
				if _, exists := mon.docs["cluster-3"]; !exists {
					t.Errorf("%s: cluster-3 should not be affected by deletion", stepName)
				}
			},
		},
		{
			name:      "delete non-existent document",
			action:    Delete,
			clusterID: "non-existent",
			bucket:    5,
			state:     api.ProvisioningStateDeleting,
			validate: func(t *testing.T, stepName string, mon *monitor) {
				if len(mon.docs) != 2 {
					t.Errorf("%s: expected 2 documents (no change), got %d", stepName, len(mon.docs))
				}
				// Verify existing documents are unaffected
				if _, exists := mon.docs["cluster-2"]; !exists {
					t.Errorf("%s: existing documents should not be affected", stepName)
				}
				if _, exists := mon.docs["cluster-3"]; !exists {
					t.Errorf("%s: existing documents should not be affected", stepName)
				}
			},
		},
		{
			name:      "test fixDoc - remove bucket ownership",
			action:    NoOp,
			clusterID: "cluster-3",
			bucket:    2,
			state:     api.ProvisioningStateSucceeded,
			validate: func(t *testing.T, stepName string, mon *monitor) {
				// First verify worker exists
				if mon.docs["cluster-3"].stop == nil {
					t.Fatalf("%s: worker should exist before ownership change", stepName)
				}

				// Remove bucket ownership
				delete(mon.buckets, 2)

				// Call fixDoc
				doc := createMockClusterDoc("cluster-3", 2, api.ProvisioningStateSucceeded)
				mon.fixDoc(doc)

				// Verify worker was stopped
				if mon.docs["cluster-3"].stop != nil {
					t.Errorf("%s: worker should be stopped when bucket no longer owned", stepName)
				}
			},
		},
	}

	// Execute operations in sequence on the same monitor
	testMon.mu.Lock()
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			doc := createMockClusterDoc(op.clusterID, op.bucket, op.state)

			switch op.action {
			case Upsert:
				testMon.upsertDoc(doc)
			case Delete:
				testMon.deleteDoc(doc)
			case NoOp:
				// Do nothing, we don't need to call any func for the test to run
			default:
				t.Fatalf("unknown test action")
			}

			if op.validate != nil {
				op.validate(t, op.name, testMon)
			}
		})
	}
	testMon.mu.Unlock()
}

func TestConcurrentUpsert(t *testing.T) {
	env := createTestEnvironmentWithLocalCosmos(t)
	defer env.LocalCosmosCleanup()

	doc := createMockClusterDoc("cluster-concurrent", 1, api.ProvisioningStateSucceeded)
	mon := env.CreateTestMonitor("cluster-concurrent")
	mon.buckets[1] = struct{}{}
	wg := sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			mon.mu.Lock()
			// Holding the lock for a random duration, up to a second
			// As we're upserting the same doc over and over, lenght should be 1
			time.Sleep(time.Duration(rand.Intn(int(time.Second))))
			mon.upsertDoc(doc)
			mon.mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()
	if len(mon.docs) != 1 {
		t.Errorf("Expected 1 doc after the same concurrent upsert, found %d", len(mon.docs))
	}
}

func TestConcurrentDeleteChannelCloseSafety(t *testing.T) {
	env := createTestEnvironmentWithLocalCosmos(t)
	defer env.LocalCosmosCleanup()

	mon := env.CreateTestMonitor("test-channel-safety")

	mon.buckets[5] = struct{}{}

	doc := createMockClusterDoc("cluster-1", 5, api.ProvisioningStateSucceeded)
	mon.mu.Lock()
	mon.upsertDoc(doc)
	mon.mu.Unlock()

	mon.mu.Lock()
	if mon.docs["cluster-1"].stop == nil {
		t.Fatal("worker should have been created")
	}
	mon.mu.Unlock()

	// Now have multiple goroutines try to delete the same document concurrently
	numGoroutines := 10
	wg := sync.WaitGroup{}
	panicChan := make(chan interface{}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				// Catch any panic from double-closing channel
				if r := recover(); r != nil {
					panicChan <- r
				}
			}()

			mon.mu.Lock()
			mon.deleteDoc(doc)
			mon.mu.Unlock()
		}()
	}

	wg.Wait()
	close(panicChan)

	// Check if any goroutine panicked
	var panics []interface{}
	for p := range panicChan {
		panics = append(panics, p)
	}

	if len(panics) > 0 {
		t.Errorf("Expected no panics from concurrent deletes, but got %d panic(s): %v",
			len(panics), panics)
	}

	mon.mu.Lock()
	if _, exists := mon.docs["cluster-1"]; exists {
		t.Error("document should have been deleted")
	}
	mon.mu.Unlock()
}

func createMockClusterDoc(clusterID string, bucket int, provisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
	resourceID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + clusterID

	return &api.OpenShiftClusterDocument{
		MissingFields: api.MissingFields{},
		ID:            clusterID,
		ResourceID:    resourceID,
		Metadata:      map[string]interface{}{},
		Key:           resourceID,
		Bucket:        bucket,
		OpenShiftCluster: &api.OpenShiftCluster{
			ID:       resourceID,
			Name:     clusterID,
			Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
			Location: "eastus",
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState:       provisioningState,
				LastProvisioningState:   api.ProvisioningStateCreating,
				FailedProvisioningState: "",
				NetworkProfile: api.NetworkProfile{
					APIServerPrivateEndpointIP: "10.0.0.1",
				},
			},
		},
	}
}
