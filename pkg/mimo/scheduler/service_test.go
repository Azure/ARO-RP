package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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
	"github.com/Azure/ARO-RP/pkg/api/util/uuid"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	testmetrics "github.com/Azure/ARO-RP/test/util/metrics"
)

// withChangefeedGauges returns assertions together with the metrics the
// service's two changefeeds emit on every poll. Tests which run the service
// need them so that the fake emitter, which requires every emitted metric to
// be accounted for, is satisfied; the values themselves are covered by the
// changefeed's own tests.
func withChangefeedGauges(assertions []testmetrics.MetricsAssertion[int64]) []testmetrics.MetricsAssertion[int64] {
	all := make([]testmetrics.MetricsAssertion[int64], 0, len(assertions)+4)
	all = append(all, assertions...)

	for _, name := range []string{"OpenShiftClusterDocument", "SubscriptionDocument"} {
		for _, metric := range []string{changefeed.MetricStaleness, changefeed.MetricConsecutiveFailures} {
			all = append(all, testmetrics.MetricsAssertion[int64]{
				MetricName: metric,
				Dimensions: map[string]string{"name": name},
				Value:      0,
			})
		}
	}

	return all
}

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
	svc.b.StopAndWait()

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
	sched := &fakeScheduler{waitOnProcess: waitFor, whileRunning: func() {
		// Verify the worker metric is incremented during the runtime
		m.AssertSingleGauge(testmetrics.MetricsAssertion[int64]{
			MetricName: "mimo.scheduler.workers.active.count",
			Dimensions: map[string]string{},
			Value:      1,
		})
	}}

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
	svc.b.StopAndWait()
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
	poolWorkers, _ := testdatabase.NewFakePoolWorkers(_env.Now, uuid.DefaultGenerator.Generate())
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
	m.AssertGauges(withChangefeedGauges([]testmetrics.MetricsAssertion[int64]{
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
	})...)
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
	poolWorkers, poolWorkersClient := testdatabase.NewFakePoolWorkers(_env.Now, uuid.DefaultGenerator.Generate())

	// Error when it tries to get the master document
	poolWorkersClient.SetError(errors.New("boom"))

	dbs := database.NewDBGroup().
		WithMaintenanceSchedules(schedules).
		WithSubscriptions(subscriptions).
		WithOpenShiftClusters(clusters).
		WithPoolWorkers(poolWorkers)

	svc := NewService(_env, log, dbs, m)
	svc.serveHealthz = false
	svc.emitHeartbeat = false
	done := make(chan struct{})

	go svc.Run(ctx, nil, done)

	// Wait for the process to stop
	<-done

	// We will have no running workers
	r.Equal(int32(0), svc.workerCount.Load())

	// lastBucketUpdate will not have been set
	r.Nil(svc.lastBucketUpdate.Load())

	m.AssertFloats()
	m.AssertGauges()

	err := testlog.AssertLoggingOutput(hook, []testlog.ExpectedLogEntry{
		{
			"level": gomega.Equal(logrus.ErrorLevel),
			"msg":   gomega.Equal("error bootstrapping master PoolWorkerDocument (not a 412): boom"),
		},
		{
			"level": gomega.Equal(logrus.ErrorLevel),
			"msg":   gomega.Equal("error in bucket worker, exiting: boom"),
		},
		{
			"level": gomega.Equal(logrus.ErrorLevel),
			"msg":   gomega.Equal("bucket worker startup failed, exiting: boom"),
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
	ourUUID := uuid.DefaultGenerator.Generate()

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
	poolWorkers, _ := testdatabase.NewFakePoolWorkers(_env.Now, ourUUID)
	dbs := database.NewDBGroup().
		WithMaintenanceSchedules(schedules).
		WithSubscriptions(subscriptions).
		WithOpenShiftClusters(clusters).
		WithPoolWorkers(poolWorkers).
		WithMaintenanceManifests(manifests)

	ownedCluster := api.ExampleOpenShiftClusterDocument()
	ownedCluster.Bucket = 1

	unownedCluster := api.ExampleOpenShiftClusterDocument()
	unownedCluster.Bucket = 22
	unownedCluster.ID = "00000000-1111-0000-0000-000000000002"
	unownedCluster.Key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename2"
	unownedCluster.OpenShiftCluster.ID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename2"
	unownedCluster.ClusterResourceGroupIDKey = "/subscriptions/00000000-1111-0000-0000-000000000000/resourcegroups/clusterresourcegroup"
	unownedCluster.ClientIDKey = "2"

	poolWorkerMasterDoc := &api.PoolWorkerDocument{
		ID:         string(api.PoolWorkerTypeMIMOScheduler),
		WorkerType: api.PoolWorkerTypeMIMOScheduler,
		PoolWorker: &api.PoolWorker{
			// We only have bucket 1, the unowned cluster will not be served
			Buckets: []string{"other", ourUUID, "other", "other"},
		},
		LeaseOwner:   "other",
		LeaseExpires: 9999999999999,
	}

	fixtures.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
	fixtures.AddOpenShiftClusterDocuments(ownedCluster, unownedCluster)
	fixtures.AddPoolWorkerDocuments(poolWorkerMasterDoc)

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
		WithPoolWorkers(poolWorkers).
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
	m.AssertGauges(withChangefeedGauges([]testmetrics.MetricsAssertion[int64]{
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
			Value: 2,
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
	})...)
}

func TestSchedulerServesBucketWhenChanges(t *testing.T) {
	r := require.New(t)
	ctx := t.Context()

	m := testmetrics.NewFakeMetricsEmitter(t)
	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ourUUID := uuid.DefaultGenerator.Generate()

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
	poolWorkers, poolWorkersClient := testdatabase.NewFakePoolWorkers(_env.Now, ourUUID)
	dbs := database.NewDBGroup().
		WithMaintenanceSchedules(schedules).
		WithSubscriptions(subscriptions).
		WithOpenShiftClusters(clusters).
		WithPoolWorkers(poolWorkers).
		WithMaintenanceManifests(manifests)

	ownedCluster := api.ExampleOpenShiftClusterDocument()
	ownedCluster.Bucket = 1

	notInInitialBucketCluster := api.ExampleOpenShiftClusterDocument()
	notInInitialBucketCluster.Bucket = 0
	notInInitialBucketCluster.ID = "00000000-1111-0000-0000-000000000002"
	notInInitialBucketCluster.Key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup2/providers/microsoft.redhatopenshift/openshiftclusters/resourcename2"
	notInInitialBucketCluster.OpenShiftCluster.ID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup2/providers/microsoft.redhatopenshift/openshiftclusters/resourcename2"
	notInInitialBucketCluster.ClusterResourceGroupIDKey = "/subscriptions/00000000-1111-0000-0000-000000000000/resourcegroups/clusterresourcegroup2"
	notInInitialBucketCluster.ClientIDKey = "2"

	poolWorkerMasterDoc := &api.PoolWorkerDocument{
		ID:         string(api.PoolWorkerTypeMIMOScheduler),
		WorkerType: api.PoolWorkerTypeMIMOScheduler,
		PoolWorker: &api.PoolWorker{
			// We only have bucket 1, the unowned cluster will not be served
			Buckets: []string{"other", ourUUID, "other", "other"},
		},
		LeaseOwner:   "other",
		LeaseExpires: 9999999999999,
	}

	fixtures.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
	fixtures.AddOpenShiftClusterDocuments(ownedCluster, notInInitialBucketCluster)
	fixtures.AddPoolWorkerDocuments(poolWorkerMasterDoc)

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
		WithPoolWorkers(poolWorkers).
		Create()
	r.NoError(err)

	svc := NewService(_env, log, dbs, m)
	svc.workerMaxStartupDelay = 0
	svc.interval = 10 * time.Millisecond
	svc.bucketRefreshInterval = 1 * time.Millisecond
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

	// Update the buckets so that the unowned cluster becomes owned
	_, err = poolWorkersClient.Replace(ctx, string(api.PoolWorkerTypeMIMOScheduler), &api.PoolWorkerDocument{
		ID:         string(api.PoolWorkerTypeMIMOScheduler),
		WorkerType: api.PoolWorkerTypeMIMOScheduler,
		PoolWorker: &api.PoolWorker{
			// Now we own buckets 0 and 1
			Buckets: []string{ourUUID, ourUUID, "other", "other"},
		},
		LeaseOwner:   "other",
		LeaseExpires: 9999999999999,
	}, &cosmosdb.Options{NoETag: true})
	r.NoError(err)

	checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
		ID:                "07070707-0707-0707-0707-070707070002",
		ClusterResourceID: strings.ToLower(notInInitialBucketCluster.OpenShiftCluster.ID),
		MaintenanceManifest: api.MaintenanceManifest{
			State:             api.MaintenanceManifestStatePending,
			MaintenanceTaskID: "0",
			CreatedBySchedule: "00000000-0000-0000-0000-000000000001",
			RunAfter:          time.Date(2026, 1, 1, 0, 15, 0, 0, time.UTC).Unix(),
			RunBefore:         time.Date(2026, 1, 1, 1, 15, 0, 0, time.UTC).Unix(),
		},
	})

	// Wait for our second created manifest
	r.EventuallyWithT(func(collect *assert.CollectT) {
		require.Empty(collect, checker.CheckMaintenanceManifests(manifestsClient))
	}, time.Second, time.Millisecond*10)

	// Close it after
	close(stop)
	<-done
	r.Equal(int32(0), svc.workerCount.Load())

	m.AssertFloats()
	m.AssertGauges(withChangefeedGauges([]testmetrics.MetricsAssertion[int64]{
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
			Value: 2,
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
		{
			MetricName: "mimo.scheduler.manifests.created",
			Dimensions: map[string]string{
				"resourceGroup":  "resourcegroup2",
				"resourceId":     strings.ToLower(notInInitialBucketCluster.OpenShiftCluster.ID),
				"subscriptionId": api.ExampleSubscriptionDocument().ID,
				"resourceName":   "resourcename2",
			},
			Value: 1,
		},
		// No running workers
		{
			MetricName: "mimo.scheduler.workers.active.count",
			Dimensions: map[string]string{},
			Value:      0,
		},
	})...)
}

func TestSchedulerDoesNotProcessConstantlyIfNoUpdates(t *testing.T) {
	testCases := []struct {
		desc               string
		delayFraction      float64
		expectedLowerBound int
		expectedUpperBound int
	}{
		{
			// The delay fraction at 0.0 should have unconditional reconciles
			// around 250ms, 500ms, 750ms, and one we probably won't run at 1s
			desc:               "delay fraction at 0.0",
			delayFraction:      0.0,
			expectedLowerBound: 4,
			expectedUpperBound: 6,
		},
		{
			desc:          "delay fraction at 0.5",
			delayFraction: 0.5,
			// The delay fraction at 0.5 should have unconditional reconciles
			// around 375, 625ms, 875ms, and one we won't run at 1125ms
			expectedLowerBound: 3,
			expectedUpperBound: 5,
		},
		{
			desc:          "delay fraction at 1.0",
			delayFraction: 1.0,
			// The delay fraction at 1.0 should have unconditional reconciles
			// around 500ms, 750ms, and one we probably won't run at 1s
			expectedLowerBound: 2,
			expectedUpperBound: 4,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
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
			poolWorkers, _ := testdatabase.NewFakePoolWorkers(_env.Now, uuid.DefaultGenerator.Generate())
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

			sched := &fakeScheduler{}

			svc := NewService(_env, log, dbs, m)
			svc.workerMaxStartupDelay = 0
			svc.interval = time.Millisecond
			svc.bucketRefreshInterval = time.Millisecond
			svc.scheduleUnconditionalReconcileInterval = 250 * time.Millisecond
			svc.schedulePollInterval = 1 * time.Millisecond
			svc.changefeedInterval = time.Millisecond
			svc.readinessDelay = time.Millisecond
			svc.serveHealthz = false
			svc.emitHeartbeat = false
			// This is called to get the random delay fraction
			svc.randFloat64 = func() float64 { return tC.delayFraction }
			svc.newScheduler = func(_ env.Interface, _ *logrus.Entry, _ metrics.Emitter, _ getCachedScheduleDocFunc, _ getClustersFunc, _ schedulerDBs) (Scheduler, error) {
				return sched, nil
			}
			stop := make(chan struct{})
			done := make(chan struct{})

			go svc.Run(ctx, stop, done)

			r.EventuallyWithT(func(collect *assert.CollectT) {
				require.Equal(collect, 1, sched.calls)
			}, time.Second, time.Millisecond)

			// Sleep for a second so that the loop will trigger the unconditional
			// reevaluate condition
			time.Sleep(time.Second)

			// The scheduler should be called according to the unconditional
			// reconcile interval because nothing has changed. Allow a range of
			// total calls to make timer delays in the goroutine or this test
			// less likely to flake -- as long as it's not hundreds or remains
			// at 1
			r.GreaterOrEqual(sched.calls, tC.expectedLowerBound)
			r.LessOrEqual(sched.calls, tC.expectedUpperBound)

			close(stop)

			// Then wait for the worker to stop
			<-done
			r.Equal(int32(0), svc.workerCount.Load())

			m.AssertFloats()
			m.AssertGauges(withChangefeedGauges([]testmetrics.MetricsAssertion[int64]{
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
			})...)
		})
	}
}

func TestShouldReevaluateUnconditionally(t *testing.T) {
	testCases := []struct {
		desc          string
		now           time.Time
		lastRan       time.Time
		interval      time.Duration
		delayFraction float64
		expectedValue bool
	}{
		{
			desc:          "every 60 mins, it's been 60 mins, no delay",
			now:           time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC),
			lastRan:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:      time.Hour,
			delayFraction: 0.0,
			expectedValue: true,
		},
		{
			desc:          "every 60 mins, it's been 59:59 mins, no delay",
			now:           time.Date(2026, 1, 1, 0, 59, 59, 0, time.UTC),
			lastRan:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:      time.Hour,
			delayFraction: 0.0,
			expectedValue: false,
		},
		{
			desc:          "every 60 mins, it's been 90 mins, 50% delay",
			now:           time.Date(2026, 1, 1, 1, 30, 0, 0, time.UTC),
			lastRan:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:      time.Hour,
			delayFraction: 0.5,
			expectedValue: true,
		},
		{
			desc:          "every 60 mins, it's been 89:59 mins, 50% delay",
			now:           time.Date(2026, 1, 1, 1, 29, 59, 0, time.UTC),
			lastRan:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:      time.Hour,
			delayFraction: 0.5,
			expectedValue: false,
		},
		{
			desc:          "every 60 mins, it's been 60 mins, 100% delay",
			now:           time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC),
			lastRan:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:      time.Hour,
			delayFraction: 1.0,
			expectedValue: false,
		},
		{
			desc:          "every 60 mins, it's been 120 mins, 100% delay",
			now:           time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
			lastRan:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:      time.Hour,
			delayFraction: 1.0,
			expectedValue: true,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			got := shouldReevaluateUnconditionally(tC.now, tC.lastRan, tC.interval, tC.delayFraction)
			require.Equal(t, tC.expectedValue, got)
		})
	}
}
