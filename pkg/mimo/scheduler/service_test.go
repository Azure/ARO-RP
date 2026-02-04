package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
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

var _ = Describe("MIMO Scheduler Service", Ordered, func() {
	var fixtures *testdatabase.Fixture
	//var checker *testdatabase.Checker
	var manifests database.MaintenanceManifests
	//var manifestsClient *cosmosdb.FakeMaintenanceManifestDocumentClient
	var clusters database.OpenShiftClusters
	//var clustersClient cosmosdb.OpenShiftClusterDocumentClient
	var schedules database.MaintenanceSchedules
	//var schedulesClient *cosmosdb.FakeMaintenanceScheduleDocumentClient
	var m metrics.Emitter

	var svc *service

	var ctx context.Context
	var cancel context.CancelFunc

	var log *logrus.Entry
	var _env env.Interface

	var controller *gomock.Controller
	var stopChan chan struct{}

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

		log = logrus.NewEntry(&logrus.Logger{
			Out:       GinkgoWriter,
			Formatter: new(logrus.TextFormatter),
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.DebugLevel,
		})

		fixtures = testdatabase.NewFixture()
		//checker = testdatabase.NewChecker()
	})

	BeforeEach(func() {
		m = newfakeMetricsEmitter()

		now := func() time.Time { return time.Unix(120, 0) }
		manifests, _ = testdatabase.NewFakeMaintenanceManifests(now)
		schedules, _ = testdatabase.NewFakeMaintenanceSchedules(now)
		clusters, _ = testdatabase.NewFakeOpenShiftClusters()

		dbg := database.NewDBGroup().WithMaintenanceManifests(manifests).WithMaintenanceSchedules(schedules).WithOpenShiftClusters(clusters)

		svc = NewService(_env, log, dbg, m, []int{1})
		svc.now = now
		svc.workerDelay = func() time.Duration { return 0 * time.Second }
		svc.serveHealthz = false
	})

	JustBeforeEach(func() {
		err := fixtures.WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests).WithMaintenanceSchedules(schedules).Create()
		Expect(err).ToNot(HaveOccurred())

		stopChan = make(chan struct{})
		DeferCleanup(func() { close(stopChan) })
		svc.startChangefeeds(ctx, stopChan)
	})

	When("schedules are polled", func() {
		BeforeEach(func() {
			fixtures.Clear()
			fixtures.AddMaintenanceScheduleDocuments(&api.MaintenanceScheduleDocument{
				ID: "00000000-0000-0000-0000-000000000000",
				MaintenanceSchedule: api.MaintenanceSchedule{
					State: api.MaintenanceScheduleStateEnabled,
				},
			}, &api.MaintenanceScheduleDocument{
				ID: "00000000-0000-0000-0000-000000000001",
				MaintenanceSchedule: api.MaintenanceSchedule{
					State: api.MaintenanceManifestStateDisabled,
				},
			})
		})

		AfterAll(func() {
			svc.b.Stop()
		})

		It("updates the available schedules", func() {
			lastGotDocs := make(map[string]*api.MaintenanceScheduleDocument)

			newOld, err := svc.poll(ctx, lastGotDocs)
			Expect(err).ToNot(HaveOccurred())

			// Contains one that we check and one that we don't
			Expect(newOld).To(HaveLen(1))
		})

		It("removes schedules if they are not included in the fetched items", func() {
			svc.b.UpsertDoc(&api.MaintenanceScheduleDocument{ID: "00000000-0000-0000-0000-000000000002"})

			lastGotDocs := make(map[string]*api.MaintenanceScheduleDocument)
			lastGotDocs["00000000-0000-0000-0000-000000000002"] = &api.MaintenanceScheduleDocument{ID: "00000000-0000-0000-0000-000000000002"}

			newOld, err := svc.poll(ctx, lastGotDocs)
			Expect(err).ToNot(HaveOccurred())

			// Contains one that we check and one that we don't
			Expect(newOld).To(HaveLen(1))
		})
	})
})
