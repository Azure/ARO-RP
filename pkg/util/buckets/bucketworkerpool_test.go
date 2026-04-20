package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type cacheTestOperation int

const (
	Upsert cacheTestOperation = iota
	Delete
	NoOp
)

func TestUpsertAndDelete(t *testing.T) {
	_, log := testlog.LogForTesting(t)
	worker := func(stop <-chan struct{}, id string) {
		<-stop
	}

	workerPool := NewBucketWorkerPool[*api.OpenShiftClusterDocument](log, worker)

	// Set owned buckets for the entire test sequence
	ownedBuckets := []int{1, 2, 5}
	workerPool.SetBuckets(ownedBuckets)

	type operation struct {
		name      string
		action    cacheTestOperation
		clusterID string
		bucket    int
		state     api.ProvisioningState
		validate  func(*testing.T, string, *bucketWorkerPool[*api.OpenShiftClusterDocument])
	}

	operations := []operation{
		{
			name:      "upsert new document in owned bucket",
			action:    Upsert,
			clusterID: "cluster-1",
			bucket:    5,
			state:     api.ProvisioningStateSucceeded,
			validate: func(t *testing.T, stepName string, pool *bucketWorkerPool[*api.OpenShiftClusterDocument]) {
				if pool.CacheSize() != 1 {
					t.Errorf("%s: expected 1 document, got %d", stepName, pool.CacheSize())
				}
				cacheDoc, exists := pool.docs.Load(strings.ToLower(rID("cluster-1")))
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
			validate: func(t *testing.T, stepName string, pool *bucketWorkerPool[*api.OpenShiftClusterDocument]) {
				if pool.CacheSize() != 2 {
					t.Errorf("%s: expected 2 documents, got %d", stepName, pool.CacheSize())
				}
				cacheDoc, exists := pool.docs.Load(strings.ToLower(rID("cluster-2")))
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
			validate: func(t *testing.T, stepName string, pool *bucketWorkerPool[*api.OpenShiftClusterDocument]) {
				if pool.CacheSize() != 2 {
					t.Errorf("%s: expected 2 documents, got %d", stepName, pool.CacheSize())
				}
				cacheDoc, exists := pool.docs.Load(strings.ToLower(rID("cluster-1")))
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
			validate: func(t *testing.T, stepName string, pool *bucketWorkerPool[*api.OpenShiftClusterDocument]) {
				if pool.CacheSize() != 3 {
					t.Errorf("%s: expected 3 documents, got %d", stepName, pool.CacheSize())
				}
				cacheDoc, exists := pool.docs.Load(strings.ToLower(rID("cluster-3")))
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
			validate: func(t *testing.T, stepName string, pool *bucketWorkerPool[*api.OpenShiftClusterDocument]) {
				if pool.CacheSize() != 2 {
					t.Errorf("%s: expected 2 documents after deletion, got %d", stepName, pool.CacheSize())
				}
				if _, exists := pool.Doc(rID("cluster-1")); exists {
					t.Errorf("%s: document should have been deleted from cache", stepName)
				}
				// Verify other documents still exist
				if _, exists := pool.Doc(rID("cluster-2")); !exists {
					t.Errorf("%s: cluster-2 should not be affected by deletion", stepName)
				}
				if _, exists := pool.Doc(rID("cluster-3")); !exists {
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
			validate: func(t *testing.T, stepName string, pool *bucketWorkerPool[*api.OpenShiftClusterDocument]) {
				if pool.CacheSize() != 2 {
					t.Errorf("%s: expected 2 documents (no change), got %d", stepName, pool.CacheSize())
				}
				// Verify existing documents are unaffected
				if _, exists := pool.Doc(rID("cluster-2")); !exists {
					t.Errorf("%s: existing documents should not be affected", stepName)
				}
				if _, exists := pool.Doc(rID("cluster-3")); !exists {
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
			validate: func(t *testing.T, stepName string, pool *bucketWorkerPool[*api.OpenShiftClusterDocument]) {
				// First verify worker exists
				cl, _ := pool.docs.Load(strings.ToLower(rID("cluster-3")))
				if cl.stop == nil {
					t.Fatalf("%s: worker should exist before ownership change", stepName)
				}

				// Remove bucket ownership
				pool.SetBuckets([]int{1, 5})

				// Call upsertDoc
				doc := createMockClusterDoc("cluster-3", 2, api.ProvisioningStateSucceeded)
				pool.UpsertDoc(doc)

				// Verify worker was stopped
				cl, _ = pool.docs.Load(strings.ToLower(rID("cluster-3")))
				if cl.stop != nil {
					t.Errorf("%s: worker should be stopped when bucket no longer owned", stepName)
				}
			},
		},
	}

	// Execute operations in sequence on the same monitor
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			doc := createMockClusterDoc(op.clusterID, op.bucket, op.state)

			switch op.action {
			case Upsert:
				workerPool.UpsertDoc(doc)
			case Delete:
				workerPool.DeleteDoc(doc)
			case NoOp:
				// Do nothing, we don't need to call any func for the test to run
			default:
				t.Fatalf("unknown test action")
			}

			if op.validate != nil {
				op.validate(t, op.name, workerPool)
			}
		})
	}
}

func TestConcurrentUpsert(t *testing.T) {
	_, log := testlog.LogForTesting(t)
	worker := func(stop <-chan struct{}, id string) {
		<-stop
	}

	workerPool := NewBucketWorkerPool[*api.OpenShiftClusterDocument](log, worker)

	doc := createMockClusterDoc("cluster-concurrent", 1, api.ProvisioningStateSucceeded)
	workerPool.SetBuckets([]int{1})
	wg := sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			// Holding the lock for a random duration, up to a second
			// As we're upserting the same doc over and over, lenght should be 1
			time.Sleep(time.Duration(rand.Intn(int(time.Second))))
			workerPool.UpsertDoc(doc)
			wg.Done()
		}()
	}
	wg.Wait()
	if workerPool.CacheSize() != 1 {
		t.Errorf("Expected 1 doc after the same concurrent upsert, found %d", workerPool.CacheSize())
	}
}

func TestConcurrentDeleteChannelCloseSafety(t *testing.T) {
	_, log := testlog.LogForTesting(t)
	worker := func(stop <-chan struct{}, id string) {
		<-stop
	}

	workerPool := NewBucketWorkerPool[*api.OpenShiftClusterDocument](log, worker)

	workerPool.SetBuckets([]int{5})

	doc := createMockClusterDoc("cluster-1", 5, api.ProvisioningStateSucceeded)
	workerPool.UpsertDoc(doc)

	cl, _ := workerPool.docs.Load(strings.ToLower(rID("cluster-1")))
	if cl.stop == nil {
		t.Fatal("worker should have been created")
	}

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

			workerPool.DeleteDoc(doc)
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

	if _, exists := workerPool.Doc(rID("cluster-1")); exists {
		t.Error("document should have been deleted")
	}
}

func rID(clusterID string) string {
	return "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + clusterID
}

func createMockClusterDoc(clusterID string, bucket int, provisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
	resourceID := rID(clusterID)

	return &api.OpenShiftClusterDocument{
		MissingFields: api.MissingFields{},
		ID:            clusterID,
		ResourceID:    resourceID,
		Metadata:      map[string]interface{}{},
		Key:           strings.ToLower(resourceID),
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
