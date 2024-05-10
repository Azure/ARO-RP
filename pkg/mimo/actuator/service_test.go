package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type fakeMetricsEmitter struct {
	Metrics map[string]int64
}

func newfakeMetricsEmitter() *fakeMetricsEmitter {
	m := make(map[string]int64)
	return &fakeMetricsEmitter{
		Metrics: m,
	}
}

func (e *fakeMetricsEmitter) EmitGauge(metricName string, metricValue int64, dimensions map[string]string) {
	e.Metrics[metricName] = metricValue
}

func (e *fakeMetricsEmitter) EmitFloat(metricName string, metricValue float64, dimensions map[string]string) {
}

var _ = Describe("MIMO Actuator Service", Ordered, func() {
	var fixtures *testdatabase.Fixture
	var checker *testdatabase.Checker
	var manifests database.MaintenanceManifests
	var manifestsClient *cosmosdb.FakeMaintenanceManifestDocumentClient
	var clusters database.OpenShiftClusters
	//var clustersClient cosmosdb.OpenShiftClusterDocumentClient
	var m metrics.Emitter

	var svc *service

	var ctx context.Context
	var cancel context.CancelFunc

	//var hook *test.Hook
	var log *logrus.Entry
	var _env env.Interface

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

		ctx, cancel = context.WithCancel(context.Background())

		_, log = testlog.New()

		m = newfakeMetricsEmitter()

		fixtures = testdatabase.NewFixture()
		checker = testdatabase.NewChecker()
	})

	BeforeEach(func() {
		now := func() time.Time { return time.Unix(120, 0) }
		manifests, manifestsClient = testdatabase.NewFakeMaintenanceManifests(now)
		clusters, _ = testdatabase.NewFakeOpenShiftClusters()

		svc = NewService(_env, log, nil, clusters, manifests, m)
		svc.now = now

	})

	JustBeforeEach(func() {
		err := fixtures.WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests).Create()
		Expect(err).ToNot(HaveOccurred())
	})

	When("clusters are polled", func() {
		BeforeEach(func() {
			fixtures.Clear()
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(clusterResourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
				},
			})
		})

		AfterAll(func() {
			svc.b.Stop()
		})

		It("updates the available clusters", func() {
			lastGotDocs := make(map[string]*api.OpenShiftClusterDocument)

			newOld, err := svc.poll(ctx, lastGotDocs)
			Expect(err).ToNot(HaveOccurred())

			Expect(newOld).To(HaveLen(1))
		})

		It("removes clusters if they are not in the doc", func() {
			svc.b.UpsertDoc(&api.OpenShiftClusterDocument{Key: clusterResourceID + "2"})

			lastGotDocs := make(map[string]*api.OpenShiftClusterDocument)
			lastGotDocs[clusterResourceID+"2"] = &api.OpenShiftClusterDocument{Key: clusterResourceID + "2"}

			newOld, err := svc.poll(ctx, lastGotDocs)
			Expect(err).ToNot(HaveOccurred())

			Expect(newOld).To(HaveLen(1))
		})
	})

	When("maintenance needs to occur", func() {
		var manifestID string

		BeforeEach(func() {
			fixtures.Clear()
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(clusterResourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
					},
				},
			})

			manifestID = manifests.NewUUID()
			manifestID2 := manifests.NewUUID()
			fixtures.AddMaintenanceManifestDocuments(
				&api.MaintenanceManifestDocument{
					ID:        manifestID,
					ClusterID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: &api.MaintenanceManifest{
						State:     api.MaintenanceManifestStatePending,
						RunBefore: 60,
						RunAfter:  0,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:        manifestID2,
					ClusterID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: &api.MaintenanceManifest{
						State:            api.MaintenanceManifestStatePending,
						RunBefore:        300,
						RunAfter:         0,
						MaintenanceSetID: "0000-0000-0001",
					},
				})

			checker.Clear()
			checker.AddMaintenanceManifestDocuments(
				&api.MaintenanceManifestDocument{
					ID:        manifestID,
					ClusterID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: &api.MaintenanceManifest{
						State:      api.MaintenanceManifestStateTimedOut,
						StatusText: "timed out at 1970-01-01 00:02:00 +0000 UTC",
						RunBefore:  60,
						RunAfter:   0,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:        manifestID2,
					ClusterID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: &api.MaintenanceManifest{
						State:            api.MaintenanceManifestStateCompleted,
						StatusText:       "ok",
						RunBefore:        300,
						RunAfter:         0,
						MaintenanceSetID: "0000-0000-0001",
					},
				},
			)
		})

		It("expires them", func() {
			// run once
			done := make(chan struct{})
			svc.pollTime = time.Second

			svc.SetTasks(map[string]tasks.TaskFunc{
				"0000-0000-0001": func(ctx context.Context, th tasks.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
					svc.stopping.Store(true)
					return api.MaintenanceManifestStateCompleted, "ok"
				},
			})

			svc.worker(done, 0*time.Second, clusterResourceID)

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})

		It("loads the full cluster document", func() {
			// run once
			done := make(chan struct{})
			svc.pollTime = time.Second

			svc.SetTasks(map[string]tasks.TaskFunc{
				"0000-0000-0001": func(ctx context.Context, th tasks.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
					// ProvisioningState is in the full document, not just the
					// ClusterID only as in the bucket worker
					Expect(oscd.OpenShiftCluster.Properties.ProvisioningState).To(Equal(api.ProvisioningStateSucceeded))

					svc.stopping.Store(true)
					return api.MaintenanceManifestStateCompleted, "ok"
				},
			})

			svc.worker(done, 0*time.Second, clusterResourceID)

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})
	})

})
