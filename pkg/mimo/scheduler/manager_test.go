package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestProcessLoop(t *testing.T) {
	uuidGeneratorManifests := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.MAINTENANCE_MANIFESTS)
	uuidGeneratorSchedules := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.MAINTENANCE_SCHEDULES)

	manifestID := uuidGeneratorManifests.Generate()
	manifestScheduleID := uuidGeneratorSchedules.Generate()
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00001111-0000-0000-0000-000000000000"
	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	clusterResourceID2 := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName2", mockSubID)

	testCases := []struct {
		desc              string
		schedule          *api.MaintenanceScheduleDocument
		desiredSchedule   *api.MaintenanceScheduleDocument
		existingManifests []*api.MaintenanceManifestDocument
		desiredManifests  []*api.MaintenanceManifestDocument
	}{
		{
			desc: "valid schedule, new manifest created (lookahead=1, scheduleAcross=0s)",
			schedule: &api.MaintenanceScheduleDocument{
				ID: manifestScheduleID,
				MaintenanceSchedule: api.MaintenanceSchedule{
					State:             api.MaintenanceScheduleStateEnabled,
					MaintenanceTaskID: api.MIMOTaskID("0"),

					Schedule:         "Mon *-*-* 00:00:00",
					LookForwardCount: 1,
					ScheduleAcross:   "0s",

					Selectors: []*api.MaintenanceScheduleSelector{
						{
							Key:      string(SelectorDataKeySubscriptionState),
							Operator: "in",
							Values:   []string{string(api.SubscriptionStateRegistered)},
						},
					},
				},
			},
			existingManifests: []*api.MaintenanceManifestDocument{},
			desiredManifests: []*api.MaintenanceManifestDocument{
				{
					ID:                manifestID,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						Priority:          0,
						// first monday in jan 2026
						RunAfter:  time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC).Unix(),
						RunBefore: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC).Add(time.Hour).Unix(),
					},
				},
			},
		},
		{
			desc: "valid schedule, existing manifest (lookahead=1, scheduleAcross=0s)",
			schedule: &api.MaintenanceScheduleDocument{
				ID: manifestScheduleID,
				MaintenanceSchedule: api.MaintenanceSchedule{
					State:             api.MaintenanceScheduleStateEnabled,
					MaintenanceTaskID: api.MIMOTaskID("0"),

					Schedule:         "Mon *-*-* 00:00:00",
					LookForwardCount: 1,
					ScheduleAcross:   "0s",

					Selectors: []*api.MaintenanceScheduleSelector{
						{
							Key:      string(SelectorDataKeySubscriptionState),
							Operator: "in",
							Values:   []string{string(api.SubscriptionStateRegistered)},
						},
					},
				},
			},
			existingManifests: []*api.MaintenanceManifestDocument{
				{
					ID:                manifestID,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						Priority:          0,
						RunAfter:          time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC).Unix(),
						RunBefore:         time.Date(2026, 1, 5, 1, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			desiredManifests: []*api.MaintenanceManifestDocument{
				{
					ID:                manifestID,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						Priority:          0,
						// first monday in jan 2026
						RunAfter:  time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC).Unix(),
						RunBefore: time.Date(2026, 1, 5, 1, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
		},
		{
			desc: "valid schedule, new manifest created (lookahead=1, scheduleAcross=1h)",
			schedule: &api.MaintenanceScheduleDocument{
				ID: manifestScheduleID,
				MaintenanceSchedule: api.MaintenanceSchedule{
					State:             api.MaintenanceScheduleStateEnabled,
					MaintenanceTaskID: api.MIMOTaskID("0"),

					Schedule:         "Mon *-*-* 00:00:00",
					LookForwardCount: 1,
					ScheduleAcross:   "1h",

					Selectors: []*api.MaintenanceScheduleSelector{
						{
							Key:      string(SelectorDataKeySubscriptionState),
							Operator: "in",
							Values:   []string{string(api.SubscriptionStateRegistered)},
						},
					},
				},
			},
			existingManifests: []*api.MaintenanceManifestDocument{},
			desiredManifests: []*api.MaintenanceManifestDocument{
				{
					ID:                manifestID,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						Priority:          0,
						// first monday in jan 2026
						RunAfter:  time.Date(2026, 1, 5, 0, 51, 15, 0, time.UTC).Unix(),
						RunBefore: time.Date(2026, 1, 5, 1, 51, 15, 0, time.UTC).Unix(),
					},
				},
			},
		},
		{
			desc: "valid schedule, existing manifest (lookahead=1, scheduleAcross=1h)",
			schedule: &api.MaintenanceScheduleDocument{
				ID: manifestScheduleID,
				MaintenanceSchedule: api.MaintenanceSchedule{
					State:             api.MaintenanceScheduleStateEnabled,
					MaintenanceTaskID: api.MIMOTaskID("0"),

					Schedule:         "Mon *-*-* 00:00:00",
					LookForwardCount: 1,
					ScheduleAcross:   "1h",

					Selectors: []*api.MaintenanceScheduleSelector{
						{
							Key:      string(SelectorDataKeySubscriptionState),
							Operator: "in",
							Values:   []string{string(api.SubscriptionStateRegistered)},
						},
					},
				},
			},
			existingManifests: []*api.MaintenanceManifestDocument{
				{
					ID:                manifestID,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						Priority:          0,
						RunAfter:          time.Date(2026, 1, 5, 0, 51, 15, 0, time.UTC).Unix(),
						RunBefore:         time.Date(2026, 1, 5, 1, 51, 15, 0, time.UTC).Unix(),
					},
				},
			},
			desiredManifests: []*api.MaintenanceManifestDocument{
				{
					ID:                manifestID,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "0",
						Priority:          0,
						// first monday in jan 2026
						RunAfter:  time.Date(2026, 1, 5, 0, 51, 15, 0, time.UTC).Unix(),
						RunBefore: time.Date(2026, 1, 5, 1, 51, 15, 0, time.UTC).Unix(),
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			require := require.New(t)
			ctx := t.Context()

			controller := gomock.NewController(nil)
			_env := mock_env.NewMockInterface(controller)

			_, log := testlog.LogForTesting(t)

			fixtures := testdatabase.NewFixture()
			checker := testdatabase.NewChecker()

			now := func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
			manifests, manifestsClient := testdatabase.NewFakeMaintenanceManifests(now)
			schedules, schedulesClient := testdatabase.NewFakeMaintenanceSchedules(now)
			clusters, _ := testdatabase.NewFakeOpenShiftClusters()
			subscriptions, _ := testdatabase.NewFakeSubscriptions()

			dbs := database.NewDBGroup().WithMaintenanceSchedules(schedules).WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests)

			subsCache := changefeed.NewSubscriptionsChangefeedCache(false)
			clusterCache := newOpenShiftClusterCache(log, subsCache, []int{1})
			stop := make(chan struct{})
			t.Cleanup(func() { close(stop) })

			a := &scheduler{
				log: log,
				env: _env,

				dbs:         dbs,
				getClusters: clusterCache.GetClusters,

				tasks: map[api.MIMOTaskID]tasks.MaintenanceTask{},
				now:   now,
			}
			a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					return nil
				},
			})

			// The cluster+subscription fixture is always the same
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key:    strings.ToLower(clusterResourceID),
				Bucket: 1,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						MaintenanceState:  api.MaintenanceStateNone,
					},
				},
			})
			fixtures.AddSubscriptionDocuments(&api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: mockTenantID,
					},
				},
			})

			// Add a cluster that does not meet our bucket requirements and so
			// won't cause any Manifests to be created
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key:    strings.ToLower(clusterResourceID2),
				Bucket: 2,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID2,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						MaintenanceState:  api.MaintenanceStateNone,
					},
				},
			})

			// Add the schedule + any existing manifests to the fixture
			fixtures.AddMaintenanceScheduleDocuments(tC.schedule)
			fixtures.AddMaintenanceManifestDocuments(tC.existingManifests...)

			// Apply the fixture
			err := fixtures.WithOpenShiftClusters(clusters).WithSubscriptions(subscriptions).WithMaintenanceManifests(manifests).WithMaintenanceSchedules(schedules).Create()
			require.NoError(err)

			// Add the desired manifests to the checker
			checker.AddMaintenanceManifestDocuments(tC.desiredManifests...)
			// If we expect a different schedule, add that to the checker,
			// otherwise we just want to make sure it hasn't changed
			if tC.desiredSchedule != nil {
				checker.AddMaintenanceScheduleDocuments(tC.desiredSchedule)
			} else {
				checker.AddMaintenanceScheduleDocuments(tC.schedule)
			}

			// fire up the changefeeds
			go changefeed.RunChangefeed(
				ctx, log.WithField("component", "subchangefeed"), subscriptions.ChangeFeed(),
				10*time.Millisecond,
				10, subsCache, stop,
			)

			// start cluster changefeed
			go changefeed.RunChangefeed(
				ctx, log.WithField("component", "clusterchangefeed"), clusters.ChangeFeed(),
				10*time.Millisecond,
				10, clusterCache, stop,
			)

			a.cachedDoc = func() (*api.MaintenanceScheduleDocument, bool) { return tC.schedule, true }

			clusterCache.initialPopulationWaitGroup.Wait()

			didWork, err := a.Process(ctx)
			require.NoError(err)
			require.True(didWork)

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			require.Empty(errs, "MaintenanceManifests don't match")

			errs = checker.CheckMaintenanceSchedules(schedulesClient)
			require.Empty(errs, "MaintenanceSchedules don't match")

			// err = testlog.AssertLoggingOutput(hook, []testlog.ExpectedLogEntry{})
			// Expect(err).ToNot(HaveOccurred())
		})
	}
}

func TestClusterHash(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockSubID2 := "00000000-0000-0000-0000-000000000002"

	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	clusterResourceID2 := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID2)

	hash := ClusterResourceIDHashToScheduleWithinPercent(clusterResourceID)
	hash2 := ClusterResourceIDHashToScheduleWithinPercent(clusterResourceID2)

	// they need to be within 0.0-1.0, and uniqueish
	require.LessOrEqual(t, hash, 1.0)
	require.GreaterOrEqual(t, hash, 0.0)
	require.LessOrEqual(t, hash2, 1.0)
	require.GreaterOrEqual(t, hash2, 0.0)

	// it should be stable, let's make sure of that
	require.InDelta(t, 0.8542921656869101, hash, 0.0000000001)
	require.Equal(t, 3075, int(PercentWithinPeriod(hash, time.Hour).Seconds()))

	require.NotEqual(t, hash, hash2)
}

func TestClusterPercentWithinPeriod(t *testing.T) {
	testCases := []struct {
		desc      string
		percent   float64
		period    time.Duration
		endPeriod time.Duration
	}{
		{
			desc:      "10% of 1 minute",
			percent:   0.1,
			period:    time.Minute,
			endPeriod: time.Second * 6,
		},
		{
			desc:      "100% of 1 minute",
			percent:   1.0,
			period:    time.Minute,
			endPeriod: time.Second * 60,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			r := PercentWithinPeriod(tC.percent, tC.period)

			assert.Equal(t, tC.endPeriod, r)
		})
	}
}
