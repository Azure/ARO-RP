package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
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
			schedules, _ := testdatabase.NewFakeMaintenanceSchedules()
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

type fakeScheduler struct {
	hasRan        bool
	waitOnProcess *sync.WaitGroup
}

func (f *fakeScheduler) AddMaintenanceTasks(_ map[api.MIMOTaskID]tasks.MaintenanceTask) {}
func (f *fakeScheduler) Process(_ context.Context) (bool, error) {
	if f.hasRan {
		return false, nil
	}
	f.hasRan = true
	f.waitOnProcess.Done()
	return true, nil
}

func TestSchedulerStoppingWholeProcess(t *testing.T) {
	require := require.New(t)
	ctx := t.Context()

	m := testmetrics.NewFakeMetricsEmitter(t)
	controller := gomock.NewController(nil)
	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().Service().Return(strings.ToLower(string(env.SERVICE_MIMO_SCHEDULER))).AnyTimes()

	_, log := testlog.LogForTesting(t)
	fixtures := testdatabase.NewFixture()
	schedules, _ := testdatabase.NewFakeMaintenanceSchedules()
	dbs := database.NewDBGroup().WithMaintenanceSchedules(schedules)

	fixtures.AddMaintenanceScheduleDocuments(&api.MaintenanceScheduleDocument{
		ID: "00000000-0000-0000-0000-000000000000",
		MaintenanceSchedule: api.MaintenanceSchedule{
			State: api.MaintenanceScheduleStateEnabled,
		},
	})

	// Apply the fixture
	err := fixtures.WithMaintenanceSchedules(schedules).Create()
	require.NoError(err)

	waitFor := &sync.WaitGroup{}
	sched := &fakeScheduler{waitOnProcess: waitFor}

	svc := NewService(_env, log, dbs, m, []int{0})
	svc.workerDelay = func() time.Duration { return 0 * time.Second }
	svc.pollTime = 1 * time.Millisecond
	svc.newScheduler = func(_ env.Interface, _ *logrus.Entry, _ metrics.Emitter, _ getCachedScheduleDocFunc, _ getClustersFunc, _ schedulerDBs, _ func() time.Time) (Scheduler, error) {
		return sched, nil
	}

	// Ensure that it has gone through the loop at least once
	waitFor.Add(1)
	_, err = svc.poll(ctx, map[string]*api.MaintenanceScheduleDocument{})
	require.NoError(err)
	waitFor.Wait()

	// Tell the whole service to stop
	svc.stopping.Store(true)
	svc.workerRoutines.Wait()

	m.AssertFloats()
	m.AssertGauges([]testmetrics.MetricsAssertion[int64]{
		{
			MetricName: "changefeed.caches.size",
			Dimensions: map[string]string{
				"service": "mimo_scheduler",
				"name":    "MaintenanceScheduleDocument",
			},
			Value: 1,
		},
		// This will go to 1 temporarily, but will end up at 0
		{
			MetricName: "mimo.scheduler.workers.active.count",
			Dimensions: map[string]string{},
			Value:      0,
		},
	}...)
}

func TestSchedulerStoppingSingleItem(t *testing.T) {
	require := require.New(t)
	ctx := t.Context()

	m := testmetrics.NewFakeMetricsEmitter(t)
	controller := gomock.NewController(nil)
	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().Service().Return(strings.ToLower(string(env.SERVICE_MIMO_SCHEDULER))).AnyTimes()

	_, log := testlog.LogForTesting(t)
	fixtures := testdatabase.NewFixture()
	schedules, _ := testdatabase.NewFakeMaintenanceSchedules()
	dbs := database.NewDBGroup().WithMaintenanceSchedules(schedules)

	fixtures.AddMaintenanceScheduleDocuments(&api.MaintenanceScheduleDocument{
		ID: "00000000-0000-0000-0000-000000000000",
		MaintenanceSchedule: api.MaintenanceSchedule{
			State: api.MaintenanceScheduleStateEnabled,
		},
	})

	// Apply the fixture
	err := fixtures.WithMaintenanceSchedules(schedules).Create()
	require.NoError(err)

	waitFor := &sync.WaitGroup{}
	sched := &fakeScheduler{waitOnProcess: waitFor}

	svc := NewService(_env, log, dbs, m, []int{0})
	svc.workerDelay = func() time.Duration { return 0 * time.Second }
	svc.pollTime = 1 * time.Millisecond
	svc.newScheduler = func(_ env.Interface, _ *logrus.Entry, _ metrics.Emitter, _ getCachedScheduleDocFunc, _ getClustersFunc, _ schedulerDBs, _ func() time.Time) (Scheduler, error) {
		return sched, nil
	}

	// Ensure that it has gone through the loop at least once
	waitFor.Add(1)
	o, err := svc.poll(ctx, map[string]*api.MaintenanceScheduleDocument{})
	require.NoError(err)
	waitFor.Wait()

	// Disable the schedule and repoll
	schedules.Patch(ctx, "00000000-0000-0000-0000-000000000000", func(msd *api.MaintenanceScheduleDocument) error {
		msd.MaintenanceSchedule.State = api.MaintenanceScheduleStateDisabled
		return nil
	})

	o1, err := svc.poll(ctx, o)
	require.NoError(err)
	require.Empty(o1)

	// Then wait for the worker to stop
	svc.workerRoutines.Wait()
	require.Equal(int32(0), svc.workers.Load())

	m.AssertFloats()
	m.AssertGauges([]testmetrics.MetricsAssertion[int64]{
		{
			MetricName: "changefeed.caches.size",
			Dimensions: map[string]string{
				"service": "mimo_scheduler",
				"name":    "MaintenanceScheduleDocument",
			},
			// This will end up as 0 because we set the schedule to inactive
			Value: 0,
		},
		// This will go to 1 temporarily, but will end up at 0
		{
			MetricName: "mimo.scheduler.workers.active.count",
			Dimensions: map[string]string{},
			Value:      0,
		},
	}...)
}
