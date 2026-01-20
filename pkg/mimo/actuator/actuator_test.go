package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

var _ = Describe("MIMO Actuator", Ordered, func() {
	var fixtures *testdatabase.Fixture
	var checker *testdatabase.Checker
	var manifests database.MaintenanceManifests
	var manifestsClient *cosmosdb.FakeMaintenanceManifestDocumentClient
	var clusters database.OpenShiftClusters
	var clustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient
	var subscriptions database.Subscriptions
	var subscriptionsClient *cosmosdb.FakeSubscriptionDocumentClient

	var a Actuator

	var ctx context.Context
	var cancel context.CancelFunc

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
		now := func() time.Time { return time.Unix(120, 0) }
		manifests, manifestsClient = testdatabase.NewFakeMaintenanceManifests(now)
		clusters, clustersClient = testdatabase.NewFakeOpenShiftClusters()
		subscriptions, subscriptionsClient = testdatabase.NewFakeSubscriptions()

		dbc := database.NewDBGroup().WithMaintenanceManifests(manifests).WithOpenShiftClusters(clusters)

		a = &actuator{
			log: log,
			env: _env,

			clusterResourceID: strings.ToLower(clusterResourceID),

			dbs: dbc,

			tasks: map[api.MIMOTaskID]tasks.MaintenanceTask{},
			now:   now,
		}
		fixtures.Clear()
		checker.Clear()

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

		// After the the fixtures are created in each test's BeforeEach, load
		// them into the database
		err := fixtures.WithOpenShiftClusters(clusters).
			WithMaintenanceManifests(manifests).
			WithSubscriptions(subscriptions).
			Create()
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
	})

	verifyDatabaseState := func() {
		GinkgoHelper()

		errs := checker.CheckMaintenanceManifests(manifestsClient)
		Expect(errs).To(BeNil(), "MaintenanceManifests don't match")

		errs = checker.CheckOpenShiftClusters(clustersClient)
		Expect(errs).To(BeNil(), "OpenShiftClusters don't match")

		errs = checker.CheckSubscriptions(subscriptionsClient)
		Expect(errs).To(BeNil(), "Subscriptions don't match")
	}

	When("old manifest", func() {
		var manifestID string

		BeforeEach(func() {
			manifestID = manifests.NewUUID()
			fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:     api.MaintenanceManifestStatePending,
					RunBefore: 60,
					RunAfter:  0,
				},
			})

			checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:      api.MaintenanceManifestStateTimedOut,
					StatusText: "timed out at 1970-01-01 00:02:00 +0000 UTC",
					RunBefore:  60,
					RunAfter:   0,
				},
			})
		})

		It("expires them", func() {
			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeFalse())

			verifyDatabaseState()
		})
	})

	When("new manifest", func() {
		var manifestID string

		BeforeEach(func() {
			manifestID = manifests.NewUUID()
			fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:             api.MaintenanceManifestStatePending,
					MaintenanceTaskID: "0",
					RunBefore:         600,
					RunAfter:          0,
				},
			})

			checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
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
		})

		It("runs them", func() {
			a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					// check that we are in progress during this
					Expect(mmd.MaintenanceManifest.State).To(Equal(api.MaintenanceManifestStateInProgress))

					th.SetResultMessage("done")
					return nil
				},
			})

			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeTrue())

			verifyDatabaseState()
		})
	})

	When("new manifest with runAfter later than now", func() {
		var manifestID string
		var manifestThatRunsID string

		BeforeEach(func() {
			manifestID = manifests.NewUUID()
			manifestThatRunsID = manifests.NewUUID()
			fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:             api.MaintenanceManifestStatePending,
					MaintenanceTaskID: "0",
					RunBefore:         1200,
					RunAfter:          600,
				},
			})
			fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestThatRunsID,
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:             api.MaintenanceManifestStatePending,
					MaintenanceTaskID: "0",
					RunBefore:         600,
					RunAfter:          0,
				},
			})

			checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
				Dequeues:          0,
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:             api.MaintenanceManifestStatePending,
					MaintenanceTaskID: "0",
					RunBefore:         1200,
					RunAfter:          600,
				},
			})
			checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestThatRunsID,
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
		})

		It("does not run the task that is not scheduled yet", func() {
			a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					// check that we are in progress during this
					Expect(mmd.MaintenanceManifest.State).To(Equal(api.MaintenanceManifestStateInProgress))

					th.SetResultMessage("done")
					return nil
				},
			})

			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeTrue())

			verifyDatabaseState()
		})
	})

	When("new manifest for a task which repeatedly fails", func() {
		var manifestID string

		BeforeEach(func() {
			manifestID = manifests.NewUUID()
			fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID: manifestID,
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

			checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
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
		})
		It("stops after retries exceeded", func() {
			a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					return mimo.TransientError(errors.New("oh no"))
				},
			})

			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeTrue())

			verifyDatabaseState()
		})
	})

	When("new manifests", func() {
		var manifestIDs []string

		BeforeEach(func() {
			manifestIDs = []string{manifests.NewUUID(), manifests.NewUUID(), manifests.NewUUID()}
			fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestIDs[0],
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
					ID:                manifestIDs[1],
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
					ID:                manifestIDs[2],
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						MaintenanceTaskID: "2",
						RunBefore:         600,
						RunAfter:          1,
						Priority:          0,
					},
				})

			checker.AddMaintenanceManifestDocuments(
				&api.MaintenanceManifestDocument{
					ID:                manifestIDs[0],
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateCompleted,
						MaintenanceTaskID: "0",
						StatusText:        "done",
						RunBefore:         600,
						RunAfter:          0,
						Priority:          2,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:                manifestIDs[1],
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateCompleted,
						MaintenanceTaskID: "1",
						StatusText:        "done",
						RunBefore:         600,
						RunAfter:          0,
						Priority:          1,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:                manifestIDs[2],
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateCompleted,
						MaintenanceTaskID: "2",
						StatusText:        "done",
						RunBefore:         600,
						RunAfter:          1,
						Priority:          0,
					},
				})
		})

		It("runs them in priority order", func() {
			ordering := []string{}

			a.AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					ordering = append(ordering, "0")
					th.SetResultMessage("done")
					return nil
				},
				"1": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					ordering = append(ordering, "1")
					th.SetResultMessage("done")
					return nil
				},
				"2": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) error {
					ordering = append(ordering, "2")
					th.SetResultMessage("done")
					return nil
				},
			})

			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeTrue())

			// We expect 1 (start time of 0, but higher priority), then 0 (start
			// time of 0, lower priority), then 2 (start time of 1, then highest
			// priority)
			Expect(ordering).To(BeEquivalentTo([]string{"1", "0", "2"}))

			verifyDatabaseState()
		})
	})

})

func TestActuator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Actuator Suite")
}
