package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/uuid"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var _ = Describe("MIMO Scheduler", Ordered, func() {
	var fixtures *testdatabase.Fixture
	var checker *testdatabase.Checker
	var subscriptions database.Subscriptions
	var schedules database.MaintenanceSchedules
	var schedulesClient *cosmosdb.FakeMaintenanceScheduleDocumentClient
	var manifests database.MaintenanceManifests
	var manifestsClient *cosmosdb.FakeMaintenanceManifestDocumentClient
	var clusters database.OpenShiftClusters
	var clustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient

	var uuidGeneratorManifests uuid.Generator

	var a *scheduler

	var ctx context.Context
	var cancel context.CancelFunc
	var stop chan struct{}

	var subsCache changefeed.SubscriptionsCache
	var clusterCache *openShiftClusterCache

	var log *logrus.Entry
	var hook *test.Hook
	var _env env.Interface

	var controller *gomock.Controller

	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00001111-0000-0000-0000-000000000000"
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

		hook, log = testlog.LogForTesting(GinkgoTB())

		fixtures = testdatabase.NewFixture()
		checker = testdatabase.NewChecker()
	})

	BeforeEach(func() {
		now := func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
		manifests, manifestsClient = testdatabase.NewFakeMaintenanceManifests(now)
		schedules, schedulesClient = testdatabase.NewFakeMaintenanceSchedules(now)
		clusters, clustersClient = testdatabase.NewFakeOpenShiftClusters()
		subscriptions, _ = testdatabase.NewFakeSubscriptions()

		dbs := database.NewDBGroup().WithMaintenanceSchedules(schedules).WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests)

		subsCache = changefeed.NewSubscriptionsChangefeedCache(false)
		clusterCache = newOpenShiftClusterCache(log, subsCache)
		stop = make(chan struct{})
		DeferCleanup(func() { close(stop) })

		a = &scheduler{
			log: log,
			env: _env,

			dbs:         dbs,
			getClusters: clusterCache.GetClusters,

			tasks: map[api.MIMOTaskID]tasks.MaintenanceTask{},
			now:   now,
		}
		fixtures.Clear()
		checker.Clear()
		hook.Reset()

		uuidGeneratorManifests = deterministicuuid.NewTestUUIDGenerator(deterministicuuid.MAINTENANCE_MANIFESTS)
	})

	JustBeforeEach(func() {
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
		fixtures.AddSubscriptionDocuments(&api.SubscriptionDocument{
			ID: mockSubID,
			Subscription: &api.Subscription{
				State: api.SubscriptionStateRegistered,
				Properties: &api.SubscriptionProperties{
					TenantID: mockTenantID,
				},
			},
		})

		// After the the fixtures are created in each test's BeforeEach, load
		// them into the database
		err := fixtures.WithOpenShiftClusters(clusters).WithSubscriptions(subscriptions).WithMaintenanceManifests(manifests).WithMaintenanceSchedules(schedules).Create()
		Expect(err).ToNot(HaveOccurred())

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
	})

	verifyDatabaseState := func() {
		GinkgoHelper()

		errs := checker.CheckMaintenanceManifests(manifestsClient)
		Expect(errs).To(BeNil(), "MaintenanceManifests don't match")

		errs = checker.CheckOpenShiftClusters(clustersClient)
		Expect(errs).To(BeNil(), "OpenShiftClusters don't match")

		errs = checker.CheckMaintenanceSchedules(schedulesClient)
		Expect(errs).To(BeNil(), "MaintenanceSchedules don't match")
	}

	When("active schedule", func() {
		var manifestScheduleID string
		var manifestID string
		var schedule *api.MaintenanceScheduleDocument

		BeforeEach(func() {
			manifestScheduleID = schedules.NewUUID()
			schedule = &api.MaintenanceScheduleDocument{
				ID: manifestScheduleID,
				MaintenanceSchedule: api.MaintenanceSchedule{
					State:             api.MaintenanceScheduleStateEnabled,
					MaintenanceTaskID: api.MIMOTaskID("0"),

					Schedule:         "Mon *-*-* 00:00:00",
					LookForwardCount: 1,
					ScheduleAcross:   "0 seconds",

					Selectors: []*api.MaintenanceScheduleSelector{
						{
							Key:      string(SelectorDataKeySubscriptionState),
							Operator: "in",
							Values:   []string{string(api.SubscriptionStateRegistered)},
						},
					},
				},
			}
			fixtures.AddMaintenanceScheduleDocuments(schedule)

			// Schedule is unchanged
			checker.AddMaintenanceScheduleDocuments(schedule)

			// first monday in jan 2026
			t := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
			manifestID = uuidGeneratorManifests.Generate()

			checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID: manifestID,

				ClusterResourceID: clusterResourceID,
				MaintenanceManifest: api.MaintenanceManifest{
					State:             api.MaintenanceManifestStatePending,
					MaintenanceTaskID: "0",
					Priority:          0,
					RunAfter:          t.Unix(),
				},
			})
		})

		It("doesn't create a manifest for active schedules when one already exists", func() {
			a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					return nil
				},
			})

			// create the expected one
			manifests.Create(ctx, &api.MaintenanceManifestDocument{
				ID: manifestID,

				ClusterResourceID: clusterResourceID,
				MaintenanceManifest: api.MaintenanceManifest{
					State:             api.MaintenanceManifestStatePending,
					MaintenanceTaskID: "0",
					Priority:          0,
					RunAfter:          time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC).Unix(),
				},
			})

			a.cachedDoc = func() (*api.MaintenanceScheduleDocument, bool) { return schedule, true }

			clusterCache.initialPopulationWaitGroup.Wait()

			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeTrue())

			verifyDatabaseState()

			// err = testlog.AssertLoggingOutput(hook, []testlog.ExpectedLogEntry{})
			// Expect(err).ToNot(HaveOccurred())
		})

		It("creates manifests for active schedules when there are none in future", func() {
			a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					return nil
				},
			})

			a.cachedDoc = func() (*api.MaintenanceScheduleDocument, bool) { return schedule, true }

			clusterCache.initialPopulationWaitGroup.Wait()

			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeTrue())

			verifyDatabaseState()

			err = testlog.AssertLoggingOutput(hook, []testlog.ExpectedLogEntry{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func TestClusterHash(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockSubID2 := "00000000-0000-0000-0000-000000000002"

	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	clusterResourceID2 := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID2)

	hash := ClusterResourceIDHashToScheduleWithinPercent(clusterResourceID)
	hash2 := ClusterResourceIDHashToScheduleWithinPercent(clusterResourceID2)

	// they need to be within 0.0-1.0, and uniqueish
	assert.LessOrEqual(t, hash, 1.0)
	assert.GreaterOrEqual(t, hash, 0.0)
	assert.LessOrEqual(t, hash2, 1.0)
	assert.GreaterOrEqual(t, hash2, 0.0)

	assert.NotEqual(t, hash, hash2)
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
