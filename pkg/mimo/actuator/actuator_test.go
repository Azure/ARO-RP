package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestActuatorLogic(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	manifestID1 := "07070707-0707-0707-0707-070707070001"
	manifestID2 := "07070707-0707-0707-0707-070707070002"
	manifestID3 := "07070707-0707-0707-0707-070707070003"

	testCases := []struct {
		desc        string
		fixtures    func(f *testdatabase.Fixture)
		checkers    func(c *testdatabase.Checker)
		tasks       func(r *require.Assertions) map[api.MIMOTaskID]tasks.MaintenanceTask
		wantDidWork bool
		wantErr     error
		wantLogs    []testlog.ExpectedLogEntry
	}{
		{
			desc: "old manifests are expired",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:     api.MaintenanceManifestStatePending,
						RunBefore: 60,
						RunAfter:  0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:      api.MaintenanceManifestStateTimedOut,
						StatusText: "timed out at 1970-01-01 00:02:00 +0000 UTC",
						RunBefore:  60,
						RunAfter:   0,
					},
				})
			},
			wantDidWork: false,
		},
		{
			desc: "manifest with runAfter in the past and runBefore in the future is run",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateCompleted,
						MaintenanceTaskID: "0",
						StatusText:        "done",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			tasks: func(r *require.Assertions) map[api.MIMOTaskID]tasks.MaintenanceTask {
				return map[api.MIMOTaskID]tasks.MaintenanceTask{
					"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						// check that we are in progress during this
						r.Equal(api.MaintenanceManifestStateInProgress, mmd.MaintenanceManifest.State)

						th.SetResultMessage("done")
						return nil
					},
				}
			},
			wantDidWork: true,
		},
		{
			desc: "no valid subscription in db",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					Dequeues:          0,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			wantDidWork: false,
			wantErr:     errFailedFetchingSubscriptionDocument,
		},
		{
			desc: "unknown task",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateFailed,
						StatusText:        "task ID not registered",
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			tasks:       func(r *require.Assertions) map[api.MIMOTaskID]tasks.MaintenanceTask { return nil },
			wantDidWork: false,
		},
		{
			desc: "new manifest with runAfter later than now does not run",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         1200,
						RunAfter:          600,
					},
				})
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID2,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					Dequeues:          0,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         1200,
						RunAfter:          600,
					},
				})
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID2,
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateCompleted,
						MaintenanceTaskID: "0",
						StatusText:        "done",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			wantDidWork: true,
		},
		{
			desc: "a task that repeatedly fails will stop after retries are exceeded",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID: manifestID1,
					// Set the dequeue count to right before it would fail
					Dequeues:          maxDequeueCount - 1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					Dequeues:          maxDequeueCount,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateRetriesExceeded,
						MaintenanceTaskID: "0",
						StatusText:        "did not succeed after 5 times, failing -- TransientError: oh no",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			tasks: func(r *require.Assertions) map[api.MIMOTaskID]tasks.MaintenanceTask {
				return map[api.MIMOTaskID]tasks.MaintenanceTask{
					"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						return mimo.TransientError(errors.New("oh no"))
					},
				}
			},
			wantDidWork: true,
		},
		{
			desc: "a task that fails the first time with a transient error will go back into pending",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			tasks: func(r *require.Assertions) map[api.MIMOTaskID]tasks.MaintenanceTask {
				return map[api.MIMOTaskID]tasks.MaintenanceTask{
					"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						return mimo.TransientError(errors.New("oh no"))
					},
				}
			},
			wantDidWork: true,
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": Equal(logrus.InfoLevel),
					"msg":   Equal("Processing 1 manifests"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("begin processing manifest"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("executing manifest"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.ErrorLevel),
					"msg":        Equal("task returned a retryable error: TransientError: oh no"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("manifest processing complete"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
			},
		},
		{
			desc: "a task that fails with a terminal error will be marked as failed",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateFailed,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			tasks: func(r *require.Assertions) map[api.MIMOTaskID]tasks.MaintenanceTask {
				return map[api.MIMOTaskID]tasks.MaintenanceTask{
					"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						return mimo.TerminalError(errors.New("oh no"))
					},
				}
			},
			wantDidWork: true,
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": Equal(logrus.InfoLevel),
					"msg":   Equal("Processing 1 manifests"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("begin processing manifest"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("executing manifest"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.ErrorLevel),
					"msg":        Equal("task returned a terminal error: TerminalError: oh no"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("manifest processing complete"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
			},
		},
		{
			desc: "a task that errors without a MIMO error wrapper is marked as failing terminally",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			checkers: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                manifestID1,
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateFailed,
						MaintenanceTaskID: "0",
						RunBefore:         600,
						RunAfter:          0,
					},
				})
			},
			tasks: func(r *require.Assertions) map[api.MIMOTaskID]tasks.MaintenanceTask {
				return map[api.MIMOTaskID]tasks.MaintenanceTask{
					"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						return errors.New("oh no")
					},
				}
			},
			wantDidWork: true,
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": Equal(logrus.InfoLevel),
					"msg":   Equal("Processing 1 manifests"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("begin processing manifest"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("executing manifest"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.ErrorLevel),
					"msg":        Equal("task returned a terminal error: oh no"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
				{
					"level":      Equal(logrus.InfoLevel),
					"msg":        Equal("manifest processing complete"),
					"taskID":     Equal("0"),
					"manifestID": Equal("07070707-0707-0707-0707-070707070001"),
				},
			},
		},
		{
			desc: "manifests are run in priority order",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(
					&api.SubscriptionDocument{
						ID: mockSubID,
					},
				)
				f.AddMaintenanceManifestDocuments(
					&api.MaintenanceManifestDocument{
						ID:                manifestID1,
						ClusterResourceID: strings.ToLower(clusterResourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							State:             api.MaintenanceManifestStatePending,
							MaintenanceTaskID: "0",
							RunBefore:         600,
							RunAfter:          0,
							Priority:          2,
						},
					},
					&api.MaintenanceManifestDocument{
						ID:                manifestID2,
						ClusterResourceID: strings.ToLower(clusterResourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							State:             api.MaintenanceManifestStatePending,
							MaintenanceTaskID: "1",
							RunBefore:         600,
							RunAfter:          0,
							Priority:          1,
						},
					},
					&api.MaintenanceManifestDocument{
						ID:                manifestID3,
						ClusterResourceID: strings.ToLower(clusterResourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							State:             api.MaintenanceManifestStatePending,
							MaintenanceTaskID: "2",
							RunBefore:         600,
							RunAfter:          1,
							Priority:          0,
						},
					},
				)
			},
			checkers: func(c *testdatabase.Checker) {
				// We expect 1 (start time of 0, but higher priority), then 0 (start
				// time of 0, lower priority), then 2 (start time of 1, then highest
				// priority)
				c.AddMaintenanceManifestDocuments(
					&api.MaintenanceManifestDocument{
						ID:                manifestID1,
						Dequeues:          1,
						ClusterResourceID: strings.ToLower(clusterResourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							State:             api.MaintenanceManifestStateCompleted,
							MaintenanceTaskID: "0",
							StatusText:        "1,0",
							RunBefore:         600,
							RunAfter:          0,
							Priority:          2,
						},
					},
					&api.MaintenanceManifestDocument{
						ID:                manifestID2,
						Dequeues:          1,
						ClusterResourceID: strings.ToLower(clusterResourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							State:             api.MaintenanceManifestStateCompleted,
							MaintenanceTaskID: "1",
							StatusText:        "1",
							RunBefore:         600,
							RunAfter:          0,
							Priority:          1,
						},
					},
					&api.MaintenanceManifestDocument{
						ID:                manifestID3,
						Dequeues:          1,
						ClusterResourceID: strings.ToLower(clusterResourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							State:             api.MaintenanceManifestStateCompleted,
							MaintenanceTaskID: "2",
							StatusText:        "1,0,2",
							RunBefore:         600,
							RunAfter:          1,
							Priority:          0,
						},
					},
				)
			},
			tasks: func(r *require.Assertions) map[api.MIMOTaskID]tasks.MaintenanceTask {
				ordering := []string{}

				return map[api.MIMOTaskID]tasks.MaintenanceTask{
					"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						ordering = append(ordering, "0")
						th.SetResultMessage(strings.Join(ordering, ","))
						return nil
					},
					"1": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						ordering = append(ordering, "1")
						th.SetResultMessage(strings.Join(ordering, ","))
						return nil
					},
					"2": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						ordering = append(ordering, "2")
						th.SetResultMessage(strings.Join(ordering, ","))
						return nil
					},
				}
			},
			wantDidWork: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			r := require.New(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time {
				return time.Unix(120, 0)
			})

			manifests, manifestsClient := testdatabase.NewFakeMaintenanceManifests(_env.Now)
			clusters, clustersClient := testdatabase.NewFakeOpenShiftClusters()
			subscriptions, _ := testdatabase.NewFakeSubscriptions()

			dbs := database.NewDBGroup().
				WithMaintenanceManifests(manifests).
				WithSubscriptions(subscriptions).
				WithOpenShiftClusters(clusters)

			hook, log := testlog.LogForTesting(t)

			fixtures := testdatabase.NewFixture()
			checker := testdatabase.NewChecker()

			// The cluster fixture is always the same
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(clusterResourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						MaintenanceState:  api.MaintenanceStateNone,
					},
				},
			})

			tt.fixtures(fixtures)

			err := fixtures.WithOpenShiftClusters(clusters).
				WithMaintenanceManifests(manifests).
				WithSubscriptions(subscriptions).Create()
			r.NoError(err)

			// Perform the test

			a := &actuator{
				log: log,
				env: _env,

				clusterResourceID: strings.ToLower(clusterResourceID),

				dbs: dbs,

				tasks: map[api.MIMOTaskID]tasks.MaintenanceTask{},

				taskRunTimeout:           time.Second,
				manifestQueryBatchLength: -1,
			}

			// Register the custom tasks if we have them
			if tt.tasks != nil {
				a.AddMaintenanceTasks(tt.tasks(r))
			} else {
				a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
					"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
						th.SetResultMessage("done")
						return nil
					},
				})
			}

			didWork, err := a.Process(t.Context())
			r.Equal(tt.wantDidWork, didWork)
			r.ErrorIs(err, tt.wantErr)

			tt.checkers(checker)

			// Also check that the OpenShiftClusterDocument does not change
			checker.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(clusterResourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						MaintenanceState:  api.MaintenanceStateNone,
					},
				},
			})

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			r.Empty(errs, "MaintenanceManifests don't match")

			errs = checker.CheckOpenShiftClusters(clustersClient)
			r.Empty(errs, "OpenShiftClusters don't match")

			if tt.wantLogs != nil {
				err = testlog.AssertLoggingOutput(hook, tt.wantLogs)
				r.NoError(err)
			}
		})
	}
}
