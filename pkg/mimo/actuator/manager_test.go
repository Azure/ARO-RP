package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	utilmimo "github.com/Azure/ARO-RP/pkg/util/mimo"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestManifestStateForError(t *testing.T) {
	terminal := errors.New("terminal")
	transient := utilmimo.TransientError(errors.New("transient"))

	manifestStateForErrorTests := []struct {
		name      string
		dequeues  int
		msg       string
		err       error
		wantState api.MaintenanceManifestState
		wantMsg   string
	}{
		{
			name:      "terminalFails",
			dequeues:  1,
			msg:       "result",
			err:       terminal,
			wantState: api.MaintenanceManifestStateFailed,
			wantMsg:   "result",
		},
		{
			name:      "transientRetries",
			dequeues:  1,
			msg:       "result",
			err:       transient,
			wantState: api.MaintenanceManifestStatePending,
			wantMsg:   "result",
		},
		{
			name:      "transientExhaustedStopsRetrying",
			dequeues:  maxDequeueCount,
			msg:       "result",
			err:       transient,
			wantState: api.MaintenanceManifestStateRetriesExceeded,
			wantMsg:   fmt.Sprintf("did not succeed after %d times, failing -- %s", maxDequeueCount, transient.Error()),
		},
		{
			name:      "terminalExhaustedStopsRetrying",
			dequeues:  maxDequeueCount + 1,
			msg:       "result",
			err:       terminal,
			wantState: api.MaintenanceManifestStateRetriesExceeded,
			wantMsg:   fmt.Sprintf("did not succeed after %d times, failing -- terminal", maxDequeueCount+1),
		},
	}

	for _, tt := range manifestStateForErrorTests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, gotMsg := manifestStateForError(tt.dequeues, tt.msg, tt.err)
			if gotState != tt.wantState {
				t.Errorf("manifestStateForError(%d, %q, %v) state = %q, want %q", tt.dequeues, tt.msg, tt.err, gotState, tt.wantState)
			}
			if gotMsg != tt.wantMsg {
				t.Errorf("manifestStateForError(%d, %q, %v) msg = %q, want %q", tt.dequeues, tt.msg, tt.err, gotMsg, tt.wantMsg)
			}
		})
	}
}

// TestProcessWhenClusterDocumentUnreadable covers the case where the cluster
// document cannot be read while manifests are queued against it, which is what
// happens when the actuator cannot decrypt it.
//
// The manifests must be dequeued, so that repeated failures are bounded by
// maxDequeueCount, and every queued manifest must be attempted, rather than the
// run being abandoned at the first one.
func TestProcessWhenClusterDocumentUnreadable(t *testing.T) {
	const mockSubID = "00000000-0000-0000-0000-000000000000"
	clusterResourceID := strings.ToLower(fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID))

	manifestIDs := []string{
		"07070707-0707-0707-0707-070707070001",
		"07070707-0707-0707-0707-070707070002",
	}

	processWhenClusterDocumentUnreadableTests := []struct {
		name           string
		dequeues       int
		wantState      api.MaintenanceManifestState
		wantStatusText string
	}{
		{
			name:           "retriesWhileAttemptsRemain",
			dequeues:       0,
			wantState:      api.MaintenanceManifestStatePending,
			wantStatusText: "TransientError: failed getting cluster document: 404 : ",
		},
		{
			name:           "givesUpOnceAttemptsAreExhausted",
			dequeues:       maxDequeueCount - 1,
			wantState:      api.MaintenanceManifestStateRetriesExceeded,
			wantStatusText: fmt.Sprintf("did not succeed after %d times, failing -- TransientError: failed getting cluster document: 404 : ", maxDequeueCount),
		},
	}

	for _, tt := range processWhenClusterDocumentUnreadableTests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			controller := gomock.NewController(t)

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time {
				return time.Unix(120, 0)
			})

			manifests, manifestsClient := testdatabase.NewFakeMaintenanceManifests(_env.Now)
			clusters, _ := testdatabase.NewFakeOpenShiftClusters()
			subscriptions, _ := testdatabase.NewFakeSubscriptions()

			fixtures := testdatabase.NewFixture()
			fixtures.AddSubscriptionDocuments(&api.SubscriptionDocument{ID: mockSubID})

			// Deliberately no OpenShiftClusterDocument, so that the actuator's
			// read of it fails in the same way as an undecryptable document.
			for _, id := range manifestIDs {
				fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                id,
					ClusterResourceID: clusterResourceID,
					Dequeues:          tt.dequeues,
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			}

			r.NoError(fixtures.WithOpenShiftClusters(clusters).
				WithMaintenanceManifests(manifests).
				WithSubscriptions(subscriptions).Create())

			_, log := testlog.LogForTesting(t)

			a := &actuator{
				log:                      log,
				env:                      _env,
				clusterResourceID:        clusterResourceID,
				dbs:                      database.NewDBGroup().WithMaintenanceManifests(manifests).WithSubscriptions(subscriptions).WithOpenShiftClusters(clusters),
				tasks:                    map[api.MIMOTaskID]tasks.MaintenanceTask{},
				taskRunTimeout:           time.Second,
				manifestQueryBatchLength: -1,
			}
			a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0": func(th utilmimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					t.Error("the task ran, but the cluster document could not be read")
					return nil
				},
			})

			// A failure to read the cluster document is not returned: it says
			// nothing about the other clusters the actuator has to service.
			didWork, err := a.Process(t.Context())
			r.NoError(err)
			r.True(didWork, "Process() reported no work, so the failure would not be visible")

			checker := testdatabase.NewChecker()
			for _, id := range manifestIDs {
				checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                id,
					ClusterResourceID: clusterResourceID,
					// Incremented by the lease, which is what bounds retries.
					Dequeues: tt.dequeues + 1,
					MaintenanceManifest: api.MaintenanceManifest{
						State:             tt.wantState,
						StatusText:        tt.wantStatusText,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			}

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			r.Empty(errs, "MaintenanceManifests don't match")
		})
	}
}
