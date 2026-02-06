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
	env := SetupTestEnvironment(t)
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

	// Give any workers time to fully exit after lock is released
	time.Sleep(200 * time.Millisecond)
}

func TestConcurrentUpsert(t *testing.T) {
	env := SetupTestEnvironment(t)
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

	// Close all workers opened by upsert
	mon.mu.Lock()
	for _, cacheDoc := range mon.docs {
		if cacheDoc.stop != nil {
			close(cacheDoc.stop)
		}
	}
	mon.mu.Unlock()
	// Give worker time to exit after stop channel was closed
	time.Sleep(200 * time.Millisecond)
}

func TestConcurrentDeleteChannelCloseSafety(t *testing.T) {
	env := SetupTestEnvironment(t)
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
	// Close any remaining workers (should be none since deleteDoc closes them)
	for _, cacheDoc := range mon.docs {
		if cacheDoc.stop != nil {
			close(cacheDoc.stop)
		}
	}
	mon.mu.Unlock()
	// Give worker time to exit after stop channel was closed
	time.Sleep(200 * time.Millisecond)
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

func TestStripUnusedFields(t *testing.T) {
	tests := []struct {
		name     string
		input    *api.OpenShiftClusterDocument
		validate func(*testing.T, *api.OpenShiftClusterDocument)
	}{
		{
			name:  "nil document returns nil",
			input: nil,
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if result != nil {
					t.Error("expected nil result for nil input")
				}
			},
		},
		{
			name: "nil OpenShiftCluster returns original",
			input: &api.OpenShiftClusterDocument{
				ID:               "test-id",
				OpenShiftCluster: nil,
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if result == nil || result.ID != "test-id" {
					t.Error("expected original document with nil OpenShiftCluster")
				}
			},
		},
		{
			name: "strips sensitive fields",
			input: &api.OpenShiftClusterDocument{
				ID:           "cluster-1",
				Key:          "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-1",
				PartitionKey: "partition-1",
				Bucket:       5,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-1",
					Name:     "cluster-1",
					Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
					Location: "eastus",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						ClusterProfile: api.ClusterProfile{
							PullSecret: "super-secret-pull-secret",
							Domain:     "test.example.com",
							Version:    "4.12.0",
						},
						NetworkProfile: api.NetworkProfile{
							APIServerPrivateEndpointIP: "10.0.0.1",
							PreconfiguredNSG:           api.PreconfiguredNSGDisabled,
						},
						MasterProfile: api.MasterProfile{
							SubnetID: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks/vnet1/subnets/master",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								Name:     "worker",
								SubnetID: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks/vnet1/subnets/worker",
								Count:    3,
								VMSize:   "Standard_D4s_v3",
							},
						},
						APIServerProfile: api.APIServerProfile{
							URL: "https://api.test.example.com:6443",
						},
						SSHKey:            api.SecureBytes("ssh-rsa AAAAB3..."),
						AdminKubeconfig:   api.SecureBytes("admin-kubeconfig-data"),
						KubeadminPassword: "admin-password",
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:     "registry1",
								Username: "user1",
								Password: "password1",
							},
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							ClientID:     "client-id-123",
							ClientSecret: "super-secret-client-secret",
						},
					},
				},
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				// Verify document metadata preserved
				if result.ID != "cluster-1" {
					t.Errorf("expected ID 'cluster-1', got %s", result.ID)
				}
				if result.Key == "" {
					t.Error("expected Key to be preserved")
				}
				if result.Bucket != 5 {
					t.Errorf("expected Bucket 5, got %d", result.Bucket)
				}

				// Verify cluster identity preserved
				oc := result.OpenShiftCluster
				if oc.ID == "" || oc.Name != "cluster-1" || oc.Location != "eastus" {
					t.Error("cluster identity fields should be preserved")
				}

				// Verify essential properties preserved
				props := oc.Properties
				if props.ProvisioningState != api.ProvisioningStateSucceeded {
					t.Error("ProvisioningState should be preserved")
				}
				if props.NetworkProfile.APIServerPrivateEndpointIP != "10.0.0.1" {
					t.Error("APIServerPrivateEndpointIP should be preserved")
				}
				if props.MasterProfile.SubnetID == "" {
					t.Error("MasterProfile.SubnetID should be preserved")
				}
				if props.APIServerProfile.URL == "" {
					t.Error("APIServerProfile.URL should be preserved")
				}

				// Verify sensitive fields stripped
				if props.ClusterProfile.PullSecret != "" {
					t.Error("PullSecret should be stripped")
				}
				if props.SSHKey != nil {
					t.Error("SSHKey should be stripped")
				}
				if props.KubeadminPassword != "" {
					t.Error("KubeadminPassword should be stripped")
				}
				if props.RegistryProfiles != nil {
					t.Error("RegistryProfiles should be stripped")
				}

				// Verify ServicePrincipalProfile has no secret
				if props.ServicePrincipalProfile == nil {
					t.Error("ServicePrincipalProfile presence should be preserved")
				} else if props.ServicePrincipalProfile.ClientSecret != "" {
					t.Error("ServicePrincipalProfile.ClientSecret should be stripped")
				} else if props.ServicePrincipalProfile.ClientID != "client-id-123" {
					t.Error("ServicePrincipalProfile.ClientID should be preserved")
				}

				// Verify worker profiles stripped of unnecessary fields
				if len(props.WorkerProfiles) != 1 {
					t.Error("WorkerProfiles should be preserved")
				} else {
					wp := props.WorkerProfiles[0]
					if wp.SubnetID == "" {
						t.Error("WorkerProfile.SubnetID should be preserved")
					}
					if wp.VMSize != "" {
						t.Error("WorkerProfile.VMSize should be stripped")
					}
				}
			},
		},
		{
			name: "prefers AROServiceKubeconfig over AdminKubeconfig",
			input: &api.OpenShiftClusterDocument{
				ID: "cluster-2",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-2",
					Name:     "cluster-2",
					Location: "westus",
					Properties: api.OpenShiftClusterProperties{
						AdminKubeconfig:      api.SecureBytes("admin-kubeconfig"),
						AROServiceKubeconfig: api.SecureBytes("aro-service-kubeconfig"),
					},
				},
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if string(result.OpenShiftCluster.Properties.AROServiceKubeconfig) != "aro-service-kubeconfig" {
					t.Error("should prefer AROServiceKubeconfig")
				}
				// AdminKubeconfig should not be separately stored
				if result.OpenShiftCluster.Properties.AdminKubeconfig != nil {
					t.Error("AdminKubeconfig should not be stored when AROServiceKubeconfig exists")
				}
			},
		},
		{
			name: "uses AdminKubeconfig when AROServiceKubeconfig is nil",
			input: &api.OpenShiftClusterDocument{
				ID: "cluster-3",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-3",
					Name:     "cluster-3",
					Location: "eastus2",
					Properties: api.OpenShiftClusterProperties{
						AdminKubeconfig:      api.SecureBytes("admin-kubeconfig-only"),
						AROServiceKubeconfig: nil,
					},
				},
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if string(result.OpenShiftCluster.Properties.AROServiceKubeconfig) != "admin-kubeconfig-only" {
					t.Error("should use AdminKubeconfig when AROServiceKubeconfig is nil")
				}
			},
		},
		{
			name: "preserves PlatformWorkloadIdentityProfile presence",
			input: &api.OpenShiftClusterDocument{
				ID: "cluster-4",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-4",
					Name:     "cluster-4",
					Location: "northeurope",
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"identity1": {
									ResourceID: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id1",
									ClientID:   "client-1",
									ObjectID:   "object-1",
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if result.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile == nil {
					t.Error("PlatformWorkloadIdentityProfile presence should be preserved")
				}
				// But the actual identities should be stripped
				if result.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities != nil {
					t.Error("PlatformWorkloadIdentities details should be stripped")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripUnusedFields(tt.input)
			tt.validate(t, result)
		})
	}
}
