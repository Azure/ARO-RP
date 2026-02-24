package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	testmetrics "github.com/Azure/ARO-RP/test/util/metrics"
)

func TestSchedulerPolling(t *testing.T) {
	testCases := []struct {
		desc             string
		schedules        []*api.MaintenanceScheduleDocument
		previousLoop     map[string]*api.MaintenanceScheduleDocument
		desiredSchedules map[string]*api.MaintenanceScheduleDocument
		expectedLogs     []testlog.ExpectedLogEntry
		expectedMetrics  []testmetrics.MetricsAssertion[int64]
	}{
		{
			desc: "schedules are polled and updated",
			schedules: []*api.MaintenanceScheduleDocument{
				{
					ID: "00000000-0000-0000-0000-000000000000",
					MaintenanceSchedule: api.MaintenanceSchedule{
						State: api.MaintenanceScheduleStateEnabled,
					},
				}, {
					ID: "00000000-0000-0000-0000-000000000001",
					MaintenanceSchedule: api.MaintenanceSchedule{
						State: api.MaintenanceScheduleStateDisabled,
					},
				},
			},
			previousLoop: map[string]*api.MaintenanceScheduleDocument{},
			desiredSchedules: map[string]*api.MaintenanceScheduleDocument{
				"00000000-0000-0000-0000-000000000000": {
					ID: "00000000-0000-0000-0000-000000000000",
					MaintenanceSchedule: api.MaintenanceSchedule{
						State: api.MaintenanceScheduleStateEnabled,
					},
				},
			},
			expectedMetrics: []testmetrics.MetricsAssertion[int64]{
				{
					MetricName: "changefeed.caches.size",
					Dimensions: map[string]string{
						"service": "mimo_scheduler",
						"name":    "MaintenanceScheduleDocument",
					},
					Value: 1,
				},
			},
		},
		{
			desc: "schedules are removed if they are not in a poll",
			schedules: []*api.MaintenanceScheduleDocument{
				{
					ID: "00000000-0000-0000-0000-000000000000",
					MaintenanceSchedule: api.MaintenanceSchedule{
						State: api.MaintenanceScheduleStateEnabled,
					},
				}, {
					ID: "00000000-0000-0000-0000-000000000001",
					MaintenanceSchedule: api.MaintenanceSchedule{
						State: api.MaintenanceScheduleStateDisabled,
					},
				},
			},
			previousLoop: map[string]*api.MaintenanceScheduleDocument{
				"00000000-0000-0000-0000-000000000002": {ID: "00000000-0000-0000-0000-000000000002"},
			},
			desiredSchedules: map[string]*api.MaintenanceScheduleDocument{
				"00000000-0000-0000-0000-000000000000": {
					ID: "00000000-0000-0000-0000-000000000000",
					MaintenanceSchedule: api.MaintenanceSchedule{
						State: api.MaintenanceScheduleStateEnabled,
					},
				},
			},
			expectedMetrics: []testmetrics.MetricsAssertion[int64]{
				{
					MetricName: "changefeed.caches.size",
					Dimensions: map[string]string{
						"service": "mimo_scheduler",
						"name":    "MaintenanceScheduleDocument",
					},
					Value: 1,
				},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			require := require.New(t)
			ctx := t.Context()

			controller := gomock.NewController(nil)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Service().Return(strings.ToLower(string(env.SERVICE_MIMO_SCHEDULER)))

			hook, log := testlog.LogForTesting(t)

			fixtures := testdatabase.NewFixture()

			now := func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
			manifests, _ := testdatabase.NewFakeMaintenanceManifests(now)
			schedules, _ := testdatabase.NewFakeMaintenanceSchedules(now)
			clusters, _ := testdatabase.NewFakeOpenShiftClusters()
			subscriptions, _ := testdatabase.NewFakeSubscriptions()

			dbs := database.NewDBGroup().WithMaintenanceSchedules(schedules).WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests)

			metrics := testmetrics.NewFakeMetricsEmitter(t)

			// Add the schedule + any existing manifests to the fixture
			fixtures.AddMaintenanceScheduleDocuments(tt.schedules...)

			// Apply the fixture
			err := fixtures.WithOpenShiftClusters(clusters).WithSubscriptions(subscriptions).WithMaintenanceManifests(manifests).WithMaintenanceSchedules(schedules).Create()
			require.NoError(err)

			svc := NewService(_env, log, dbs, metrics, []int{0})
			svc.now = now
			svc.workerDelay = func() time.Duration { return 0 * time.Second }
			svc.serveHealthz = false
			svc.stopping.Store(true)

			newOld, err := svc.poll(ctx, tt.previousLoop)
			require.NoError(err)

			diff := deep.Equal(tt.desiredSchedules, newOld)
			require.Empty(diff, "poll returned wrong results")

			err = testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			require.NoError(err)

			// check the metrics -- we don't want any floats, but we do have gauges
			metrics.AssertFloats()
			metrics.AssertGauges(tt.expectedMetrics...)
		})
	}
}
