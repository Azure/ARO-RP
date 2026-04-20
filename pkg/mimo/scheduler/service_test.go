package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
						"name": "MaintenanceScheduleDocument",
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
						"name": "MaintenanceScheduleDocument",
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
			_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })
			hook, log := testlog.LogForTesting(t)

			fixtures := testdatabase.NewFixture()

			manifests, _ := testdatabase.NewFakeMaintenanceManifests(_env.Now)
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

			svc := NewService(_env, log, dbs, metrics)
			svc.workerMaxStartupDelay = 0
			svc.serveHealthz = false
			svc.emitHeartbeat = false
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
	_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })

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

	svc := NewService(_env, log, dbs, m)
	svc.workerMaxStartupDelay = 0
	svc.interval = time.Millisecond
	svc.schedulePollInterval = 1 * time.Millisecond
	svc.newScheduler = func(_ env.Interface, _ *logrus.Entry, _ metrics.Emitter, _ getCachedScheduleDocFunc, _ getClustersFunc, _ schedulerDBs) (Scheduler, error) {
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
				"name": "MaintenanceScheduleDocument",
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
	_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })

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

	svc := NewService(_env, log, dbs, m)
	svc.workerMaxStartupDelay = 0
	svc.interval = time.Millisecond
	svc.schedulePollInterval = 1 * time.Millisecond
	svc.newScheduler = func(_ env.Interface, _ *logrus.Entry, _ metrics.Emitter, _ getCachedScheduleDocFunc, _ getClustersFunc, _ schedulerDBs) (Scheduler, error) {
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
	require.Equal(int32(0), svc.workerCount.Load())

	m.AssertFloats()
	m.AssertGauges([]testmetrics.MetricsAssertion[int64]{
		{
			MetricName: "changefeed.caches.size",
			Dimensions: map[string]string{
				"name": "MaintenanceScheduleDocument",
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

func TestSchedulerGoesReady(t *testing.T) {
	r := require.New(t)
	ctx := t.Context()

	m := testmetrics.NewFakeMetricsEmitter(t)
	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().Now().AnyTimes().DoAndReturn(time.Now)

	_, log := testlog.LogForTesting(t)
	fixtures := testdatabase.NewFixture()
	schedules, _ := testdatabase.NewFakeMaintenanceSchedules()
	clusters, _ := testdatabase.NewFakeOpenShiftClusters()
	subscriptions, _ := testdatabase.NewFakeSubscriptions()
	poolWorkers, _ := testdatabase.NewFakePoolWorkers(_env.Now)
	dbs := database.NewDBGroup().
		WithMaintenanceSchedules(schedules).
		WithSubscriptions(subscriptions).
		WithOpenShiftClusters(clusters).
		WithPoolWorkers(poolWorkers)

	fixtures.AddMaintenanceScheduleDocuments(&api.MaintenanceScheduleDocument{
		ID: "00000000-0000-0000-0000-000000000000",
		MaintenanceSchedule: api.MaintenanceSchedule{
			State: api.MaintenanceScheduleStateEnabled,
		},
	})

	// Apply the fixture
	err := fixtures.WithMaintenanceSchedules(schedules).
		WithOpenShiftClusters(clusters).
		WithSubscriptions(subscriptions).
		Create()
	r.NoError(err)

	waitFor := &sync.WaitGroup{}
	sched := &fakeScheduler{waitOnProcess: waitFor}
	waitFor.Add(1)

	svc := NewService(_env, log, dbs, m)
	svc.workerMaxStartupDelay = 0
	svc.interval = time.Millisecond
	svc.schedulePollInterval = 1 * time.Millisecond
	svc.changefeedInterval = time.Millisecond
	svc.readinessDelay = time.Millisecond
	svc.serveHealthz = false
	svc.emitHeartbeat = false
	svc.newScheduler = func(_ env.Interface, _ *logrus.Entry, _ metrics.Emitter, _ getCachedScheduleDocFunc, _ getClustersFunc, _ schedulerDBs) (Scheduler, error) {
		return sched, nil
	}
	stop := make(chan struct{})
	done := make(chan struct{})

	go svc.Run(ctx, stop, done)

	r.EventuallyWithT(func(collect *assert.CollectT) {
		require.True(collect, svc.checkReady())
	}, time.Second, time.Millisecond)

	// Wait for at least one run, and then close
	waitFor.Wait()

	close(stop)

	// Then wait for the worker to stop
	<-done
	r.Equal(int32(0), svc.workerCount.Load())

	m.AssertFloats()
	m.AssertGauges([]testmetrics.MetricsAssertion[int64]{
		{
			MetricName: "changefeed.caches.size",
			Dimensions: map[string]string{
				"name": "MaintenanceScheduleDocument",
			},
			Value: 1,
		},
		// No running workers
		{
			MetricName: "mimo.scheduler.workers.active.count",
			Dimensions: map[string]string{},
			Value:      0,
		},
	}...)
}

func TestSchedulerStopsIfBucketFailure(t *testing.T) {
	r := require.New(t)
	ctx := t.Context()

	m := testmetrics.NewFakeMetricsEmitter(t)
	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().Now().AnyTimes().DoAndReturn(time.Now)

	hook, log := testlog.LogForTesting(t)
	schedules, _ := testdatabase.NewFakeMaintenanceSchedules()
	clusters, _ := testdatabase.NewFakeOpenShiftClusters()
	subscriptions, _ := testdatabase.NewFakeSubscriptions()
	poolWorkers, poolWorkersClient := testdatabase.NewFakePoolWorkers(_env.Now)

	// Error when it tries to get the master document
	poolWorkersClient.SetError(errors.New("boom"))

	dbs := database.NewDBGroup().
		WithMaintenanceSchedules(schedules).
		WithSubscriptions(subscriptions).
		WithOpenShiftClusters(clusters).
		WithPoolWorkers(poolWorkers)

	svc := NewService(_env, log, dbs, m)
	svc.schedulePollInterval = time.Millisecond
	svc.serveHealthz = false
	svc.emitHeartbeat = false
	done := make(chan struct{})

	go svc.Run(ctx, nil, done)

	// Wait for the process to stop
	<-done

	// We will have no running workers
	r.Equal(int32(0), svc.workerCount.Load())

	m.AssertFloats()
	m.AssertGauges([]testmetrics.MetricsAssertion[int64]{
		{
			MetricName: "changefeed.caches.size",
			Dimensions: map[string]string{
				"name": "MaintenanceScheduleDocument",
			},
			Value: 0,
		},
	}...)

	err := testlog.AssertLoggingOutput(hook, []testlog.ExpectedLogEntry{
		{
			"level": gomega.Equal(logrus.ErrorLevel),
			"msg":   gomega.Equal("error bootstrapping master PoolWorkerDocument (not a 412): boom"),
		},
		{
			"level": gomega.Equal(logrus.ErrorLevel),
			"msg":   gomega.Equal("unable to start bucket worker, exiting: boom"),
		},
		{
			"level": gomega.Equal(logrus.InfoLevel),
			"msg":   gomega.Equal("exiting, waiting for all workers to finish"),
		},
	})
	r.NoError(err)
}

func TestSchedulerServesBucket(t *testing.T) {
	r := require.New(t)
	ctx := t.Context()

	m := testmetrics.NewFakeMetricsEmitter(t)
	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time {
		now = now.Add(time.Millisecond)
		return now
	})

	_, log := testlog.LogForTesting(t)
	fixtures := testdatabase.NewFixture()
	checker := testdatabase.NewChecker()
	manifests, manifestsClient := testdatabase.NewFakeMaintenanceManifests(_env.Now)
	schedules, _ := testdatabase.NewFakeMaintenanceSchedules()
	clusters, _ := testdatabase.NewFakeOpenShiftClusters()
	subscriptions, _ := testdatabase.NewFakeSubscriptions()
	poolWorkers, _ := testdatabase.NewFakePoolWorkers(_env.Now)
	dbs := database.NewDBGroup().
		WithMaintenanceSchedules(schedules).
		WithSubscriptions(subscriptions).
		WithOpenShiftClusters(clusters).
		WithPoolWorkers(poolWorkers).
		WithMaintenanceManifests(manifests)

	fixtures.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
	fixtures.AddOpenShiftClusterDocuments(api.ExampleOpenShiftClusterDocument())

	fixtures.AddMaintenanceScheduleDocuments(&api.MaintenanceScheduleDocument{
		ID: "00000000-0000-0000-0000-000000000001",
		MaintenanceSchedule: api.MaintenanceSchedule{
			State:             api.MaintenanceScheduleStateEnabled,
			MaintenanceTaskID: api.MIMOTaskID("0"),

			Schedule:         "*-*-* *:15",
			ScheduleAcross:   "0s",
			LookForwardCount: 1,

			Selectors: []*api.MaintenanceScheduleSelector{
				{
					Key:      string(SelectorDataKeySubscriptionState),
					Operator: api.MaintenanceScheduleSelectorOperatorEq,
					Value:    "Registered",
				},
			},
		},
	})

	checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
		ID:                "07070707-0707-0707-0707-070707070001",
		ClusterResourceID: strings.ToLower(api.ExampleOpenShiftClusterDocument().OpenShiftCluster.ID),
		MaintenanceManifest: api.MaintenanceManifest{
			State:             api.MaintenanceManifestStatePending,
			MaintenanceTaskID: "0",
			CreatedBySchedule: "00000000-0000-0000-0000-000000000001",
			RunAfter:          time.Date(2026, 1, 1, 0, 15, 0, 0, time.UTC).Unix(),
			RunBefore:         time.Date(2026, 1, 1, 1, 15, 0, 0, time.UTC).Unix(),
		},
	})

	// Apply the fixture
	err := fixtures.WithMaintenanceSchedules(schedules).
		WithOpenShiftClusters(clusters).
		WithSubscriptions(subscriptions).
		Create()
	r.NoError(err)

	svc := NewService(_env, log, dbs, m)
	svc.workerMaxStartupDelay = 0
	svc.interval = 10 * time.Millisecond
	svc.schedulePollInterval = 1 * time.Millisecond
	svc.changefeedInterval = time.Millisecond
	svc.readinessDelay = time.Millisecond
	svc.serveHealthz = false
	svc.emitHeartbeat = false

	stop := make(chan struct{})
	done := make(chan struct{})

	go svc.Run(ctx, stop, done)

	r.EventuallyWithT(func(collect *assert.CollectT) {
		require.True(collect, svc.checkReady())
	}, time.Second, time.Millisecond)

	// Wait for our created manifest
	r.EventuallyWithT(func(collect *assert.CollectT) {
		require.Empty(collect, checker.CheckMaintenanceManifests(manifestsClient))
	}, time.Second, time.Millisecond*10)

	// Close it after
	close(stop)
	<-done
	r.Equal(int32(0), svc.workerCount.Load())

	m.AssertFloats()
	m.AssertGauges([]testmetrics.MetricsAssertion[int64]{
		{
			MetricName: "changefeed.caches.size",
			Dimensions: map[string]string{
				"name": "MaintenanceScheduleDocument",
			},
			Value: 1,
		},
		{
			MetricName: "changefeed.caches.size",
			Dimensions: map[string]string{
				"name": "OpenShiftClusterDocument",
			},
			Value: 1,
		},
		{
			MetricName: "changefeed.caches.size",
			Dimensions: map[string]string{
				"name": "SubscriptionDocument",
			},
			Value: 1,
		},
		{
			MetricName: "mimo.scheduler.manifests.created",
			Dimensions: map[string]string{
				"resourceGroup":  "resourcegroup",
				"resourceId":     strings.ToLower(api.ExampleOpenShiftClusterDocument().OpenShiftCluster.ID),
				"subscriptionId": api.ExampleSubscriptionDocument().ID,
				"resourceName":   "resourcename",
			},
			Value: 1,
		},
		// No running workers
		{
			MetricName: "mimo.scheduler.workers.active.count",
			Dimensions: map[string]string{},
			Value:      0,
		},
	}...)
}
