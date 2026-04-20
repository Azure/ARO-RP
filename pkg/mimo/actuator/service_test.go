package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	testmetrics "github.com/Azure/ARO-RP/test/util/metrics"
)

type fakeMetricsEmitter struct {
	Metrics map[string]int64
	m       sync.RWMutex
}

func newfakeMetricsEmitter() *fakeMetricsEmitter {
	m := make(map[string]int64)
	return &fakeMetricsEmitter{
		Metrics: m,
		m:       sync.RWMutex{},
	}
}

func (e *fakeMetricsEmitter) EmitGauge(metricName string, metricValue int64, dimensions map[string]string) {
	e.m.Lock()
	defer e.m.Unlock()
	e.Metrics[metricName] = metricValue
}

func (e *fakeMetricsEmitter) EmitFloat(metricName string, metricValue float64, dimensions map[string]string) {
}

func TestActuatorPolling(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	testCases := []struct {
		desc            string
		docs            []*api.OpenShiftClusterDocument
		previousLoop    map[string]*api.OpenShiftClusterDocument
		desiredDocs     map[string]*api.OpenShiftClusterDocument
		expectedLogs    []testlog.ExpectedLogEntry
		expectedMetrics []testmetrics.MetricsAssertion[int64]
	}{
		{
			desc: "clusters are polled and updated",
			docs: []*api.OpenShiftClusterDocument{
				{
					Key:    strings.ToLower(clusterResourceID),
					Bucket: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: clusterResourceID,
					},
				},
				{
					Key:    strings.ToLower(clusterResourceID + "ignored"),
					Bucket: 2,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: clusterResourceID + "ignored",
					},
				},
			},
			previousLoop: map[string]*api.OpenShiftClusterDocument{},
			desiredDocs: map[string]*api.OpenShiftClusterDocument{
				strings.ToLower(clusterResourceID): {
					// Only essential metadata is actually stored
					Key:    strings.ToLower(clusterResourceID),
					Bucket: 1,
				},
				strings.ToLower(clusterResourceID + "ignored"): {
					Key:    strings.ToLower(clusterResourceID + "ignored"),
					Bucket: 2,
				},
			},
			expectedMetrics: []testmetrics.MetricsAssertion[int64]{
				{
					MetricName: "changefeed.caches.size",
					Dimensions: map[string]string{
						"name": "OpenShiftClusterDocument",
					},
					// we still keep clusters that aren't in our bucket in the
					// cache, in case the buckets change
					Value: 2,
				},
			},
		},
		{
			desc: "clusters are removed if they are not in a poll",
			docs: []*api.OpenShiftClusterDocument{
				{
					Key:    strings.ToLower(clusterResourceID),
					Bucket: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: clusterResourceID,
					},
				},
			},
			previousLoop: map[string]*api.OpenShiftClusterDocument{
				strings.ToLower(clusterResourceID): {
					Key:    strings.ToLower(clusterResourceID),
					Bucket: 1,
				},
				strings.ToLower(clusterResourceID + "ignored"): {
					Key:    strings.ToLower(clusterResourceID + "ignored"),
					Bucket: 2,
				},
			},
			desiredDocs: map[string]*api.OpenShiftClusterDocument{
				strings.ToLower(clusterResourceID): {
					Key:    strings.ToLower(clusterResourceID),
					Bucket: 1,
				},
			},
			expectedMetrics: []testmetrics.MetricsAssertion[int64]{
				{
					MetricName: "changefeed.caches.size",
					Dimensions: map[string]string{
						"name": "OpenShiftClusterDocument",
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

			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time {
				return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			})
			hook, log := testlog.LogForTesting(t)

			fixtures := testdatabase.NewFixture()

			manifests, _ := testdatabase.NewFakeMaintenanceManifests(_env.Now)
			clusters, _ := testdatabase.NewFakeOpenShiftClusters()
			subscriptions, _ := testdatabase.NewFakeSubscriptions()

			dbs := database.NewDBGroup().WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests).WithSubscriptions(subscriptions)

			metrics := testmetrics.NewFakeMetricsEmitter(t)

			// Apply the fixture
			fixtures.AddOpenShiftClusterDocuments(tt.docs...)
			fixtures.AddSubscriptionDocuments(&api.SubscriptionDocument{ID: mockSubID})
			err := fixtures.WithOpenShiftClusters(clusters).WithSubscriptions(subscriptions).WithMaintenanceManifests(manifests).Create()
			require.NoError(err)

			svc := NewService(_env, log, nil, dbs, metrics)
			svc.workerMaxStartupDelay = 0 * time.Second
			svc.serveHealthz = false
			svc.stopping.Store(true)

			newOld, err := svc.poll(ctx, tt.previousLoop)
			require.NoError(err)

			diff := deep.Equal(tt.desiredDocs, newOld)
			for _, e := range diff {
				t.Error(e)
			}

			err = testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			require.NoError(err)

			// check the metrics -- we don't want any floats, but we do have gauges
			metrics.AssertFloats()
			metrics.AssertGauges(tt.expectedMetrics...)
		})
	}
}

var _ = Describe("MIMO Actuator Service", Ordered, func() {
	var fixtures *testdatabase.Fixture
	var checker *testdatabase.Checker
	var manifests database.MaintenanceManifests
	var manifestsClient *cosmosdb.FakeMaintenanceManifestDocumentClient
	var clusters database.OpenShiftClusters
	var subscriptions database.Subscriptions
	var m metrics.Emitter

	var svc *service

	var ctx context.Context
	var cancel context.CancelFunc

	var log *logrus.Entry
	var _env *mock_env.MockInterface

	var controller *gomock.Controller

	mockSubID := "00000000-0000-0000-0000-000000000000"
	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	AfterAll(func() {
		if cancel != nil {
			cancel()
		}

		if controller != nil {
			controller.Finish()
		}
	})

	BeforeAll(func() {
		controller = gomock.NewController(nil)
		_env = mock_env.NewMockInterface(controller)
		_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time {
			return time.Unix(120, 0)
		})
		ctx, cancel = context.WithCancel(context.Background())

		log = logrus.NewEntry(&logrus.Logger{
			Out:       GinkgoWriter,
			Formatter: new(logrus.TextFormatter),
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.DebugLevel,
		})

		fixtures = testdatabase.NewFixture()
		checker = testdatabase.NewChecker()
	})

	BeforeEach(func() {
		m = newfakeMetricsEmitter()

		manifests, manifestsClient = testdatabase.NewFakeMaintenanceManifests(_env.Now)
		clusters, _ = testdatabase.NewFakeOpenShiftClusters()
		subscriptions, _ = testdatabase.NewFakeSubscriptions()
		dbg := database.NewDBGroup().WithMaintenanceManifests(manifests).WithOpenShiftClusters(clusters).WithSubscriptions(subscriptions)

		svc = NewService(_env, log, nil, dbg, m)
		svc.workerMaxStartupDelay = time.Second * 0
		svc.serveHealthz = false
		svc.b.SetBuckets([]int{1})
	})

	JustBeforeEach(func() {
		err := fixtures.WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests).WithSubscriptions(subscriptions).Create()
		Expect(err).ToNot(HaveOccurred())
	})

	When("maintenance needs to occur", func() {
		var manifestID string

		BeforeEach(func() {
			fixtures.Clear()
			fixtures.AddSubscriptionDocuments(
				&api.SubscriptionDocument{
					ID: mockSubID,
				},
			)
			fixtures.AddOpenShiftClusterDocuments(
				&api.OpenShiftClusterDocument{
					Key:    strings.ToLower(clusterResourceID),
					Bucket: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: clusterResourceID,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							NetworkProfile: api.NetworkProfile{
								PodCIDR: "0.0.0.0/32",
							},
						},
					},
				},
				// Cluster that will not be served because we are only looking at
				// bucket 1
				&api.OpenShiftClusterDocument{
					Key:    strings.ToLower(clusterResourceID + "ignored"),
					Bucket: 2,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: clusterResourceID + "ignored",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							NetworkProfile: api.NetworkProfile{
								PodCIDR: "0.0.0.0/32",
							},
						},
					},
				},
			)

			manifestID = manifests.NewUUID()
			manifestID2 := manifests.NewUUID()
			manifestID3 := manifests.NewUUID()

			fixtures.AddMaintenanceManifestDocuments(
				&api.MaintenanceManifestDocument{
					ID:                manifestID,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:     api.MaintenanceManifestStatePending,
						RunBefore: 60,
						RunAfter:  0,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:                manifestID2,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						RunBefore:         300,
						RunAfter:          0,
						MaintenanceTaskID: "0000-0000-0001",
					},
				},
				// A manifest for a cluster that is not served by our bucket allocation
				&api.MaintenanceManifestDocument{
					ID:                manifestID3,
					ClusterResourceID: strings.ToLower(clusterResourceID + "ignored"),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						RunBefore:         300,
						RunAfter:          0,
						MaintenanceTaskID: "0000-0000-0001",
					},
				},
			)

			checker.Clear()
			checker.AddMaintenanceManifestDocuments(
				&api.MaintenanceManifestDocument{
					ID:                manifestID,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:      api.MaintenanceManifestStateTimedOut,
						StatusText: "timed out at 1970-01-01 00:02:00 +0000 UTC",
						RunBefore:  60,
						RunAfter:   0,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:                manifestID2,
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateCompleted,
						StatusText:        "ok",
						RunBefore:         300,
						RunAfter:          0,
						MaintenanceTaskID: "0000-0000-0001",
					},
				},
				// manifest will not be served
				&api.MaintenanceManifestDocument{
					ID:                manifestID3,
					ClusterResourceID: strings.ToLower(clusterResourceID + "ignored"),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						RunBefore:         300,
						RunAfter:          0,
						MaintenanceTaskID: "0000-0000-0001",
					},
				},
			)
		})

		It("expires them", func() {
			svc.taskPollTime = time.Millisecond

			svc.SetMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0000-0000-0001": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					// once we've run this task, stop the worker
					svc.stopping.Store(true)
					th.SetResultMessage("ok")
					return nil
				},
			})

			_, err := svc.poll(ctx, nil)
			Expect(err).ToNot(HaveOccurred())

			// Wait for all of the workers to have stopped
			svc.b.WaitForWorkerCompletion()

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})

		It("loads the full cluster document", func() {
			svc.taskPollTime = time.Millisecond

			svc.SetMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0000-0000-0001": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					// Only the ClusterResourceID is available to the bucket
					// worker, so make sure this is the full document
					Expect(oscd.OpenShiftCluster.Properties.NetworkProfile.PodCIDR).To(Equal("0.0.0.0/32"))

					svc.stopping.Store(true)
					th.SetResultMessage("ok")
					return nil
				},
			})

			_, err := svc.poll(ctx, nil)
			Expect(err).ToNot(HaveOccurred())

			// Wait for all of the workers to have stopped
			svc.b.WaitForWorkerCompletion()

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})
	})
})
