package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_proxy "github.com/Azure/ARO-RP/pkg/util/mocks/proxy"
	"github.com/Azure/ARO-RP/test/util/testliveconfig"
)

func TestUpsertAndDelete(t *testing.T) {
	// Setup single monitor for all test operations
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testlogger := logrus.NewEntry(logrus.StandardLogger())
	testlogger.Logger.SetLevel(logrus.DebugLevel)
	dialer := mock_proxy.NewMockDialer(ctrl)
	mockEnv := mock_env.NewMockInterface(ctrl)
	mockEnv.EXPECT().LiveConfig().Return(testliveconfig.NewTestLiveConfig(false, false)).AnyTimes()
	noopMetricsEmitter := noop.Noop{}
	noopClusterMetricsEmitter := noop.Noop{}

	dbs := database.NewDBGroup()
	testMon := NewMonitor(testlogger, dialer, dbs, &noopMetricsEmitter, &noopClusterMetricsEmitter, mockEnv).(*monitor)

	// Set owned buckets for the entire test sequence
	ownedBuckets := []int{1, 2, 5}
	for _, bucket := range ownedBuckets {
		testMon.buckets[bucket] = struct{}{}
	}

	type operation struct {
		name      string
		action    string // "upsert", "delete", "fixDoc", "validate"
		clusterID string
		bucket    int
		state     api.ProvisioningState
		validate  func(*testing.T, string, *monitor)
	}

	operations := []operation{
		{
			name:      "upsert new document in owned bucket",
			action:    "upsert",
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
			action:    "upsert",
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
			action:    "upsert",
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
			action:    "upsert",
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
			action:    "delete",
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
			action:    "delete",
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
			action:    "fixDoc",
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
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			doc := createMockClusterDoc(op.clusterID, op.bucket, op.state)

			switch op.action {
			case "upsert":
				testMon.upsertDoc(doc)
			case "delete":
				testMon.deleteDoc(doc)
			case "fixDoc":
				// fixDoc validation is handled inside the validate function
				// because it needs to manipulate state before calling fixDoc
			default:
				t.Fatalf("unknown action: %s", op.action)
			}

			if op.validate != nil {
				op.validate(t, op.name, testMon)
			}
		})
	}
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
