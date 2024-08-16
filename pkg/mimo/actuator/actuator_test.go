package actuator

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

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/sets"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var _ = Describe("MIMO Actuator", Ordered, func() {
	var fixtures *testdatabase.Fixture
	var checker *testdatabase.Checker
	var manifests database.MaintenanceManifests
	var manifestsClient *cosmosdb.FakeMaintenanceManifestDocumentClient
	var clusters database.OpenShiftClusters
	//var clustersClient cosmosdb.OpenShiftClusterDocumentClient

	var a Actuator

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

		fixtures = testdatabase.NewFixture()
		checker = testdatabase.NewChecker()
	})

	BeforeEach(func() {
		now := func() time.Time { return time.Unix(120, 0) }
		manifests, manifestsClient = testdatabase.NewFakeMaintenanceManifests(now)
		clusters, _ = testdatabase.NewFakeOpenShiftClusters()

		a = &actuator{
			log:   log,
			env:   _env,
			mmf:   manifests,
			tasks: map[string]TaskFunc{},
			now:   now,
			log:   log,
			env:   _env,

			clusterResourceID: strings.ToLower(clusterResourceID),

			mmf: manifests,
			oc:  clusters,

			sets: map[string]sets.MaintenanceSet{},
			now:  now,
		}
	})

	JustBeforeEach(func() {
		err := fixtures.WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests).Create()
		Expect(err).ToNot(HaveOccurred())
	})

	When("old manifest", func() {
		var manifestID string

		BeforeEach(func() {
			fixtures.Clear()
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(clusterResourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
				},
			})

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

			checker.Clear()
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

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})
	})

	When("new manifest", func() {
		var manifestID string

		BeforeEach(func() {
			fixtures.Clear()
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(clusterResourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
				},
			})

			manifestID = manifests.NewUUID()
			fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:            api.MaintenanceManifestStatePending,
					MaintenanceSetID: "0",
					RunBefore:        600,
					RunAfter:         0,
				},
			})

			checker.Clear()
			checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestID,
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:            api.MaintenanceManifestStateCompleted,
					MaintenanceSetID: "0",
					StatusText:       "done",
					RunBefore:        600,
					RunAfter:         0,
				},
			})
		})

		It("runs them", func() {
			a.AddMaintenanceSets(map[string]sets.MaintenanceSet{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
					return api.MaintenanceManifestStateCompleted, "done"
				},
			})

			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeTrue())

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})
	})

	When("new manifests", func() {
		var manifestIDs []string

		BeforeEach(func() {
			fixtures.Clear()
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(clusterResourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
				},
			})

			manifestIDs = []string{manifests.NewUUID(), manifests.NewUUID(), manifests.NewUUID()}
			fixtures.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestIDs[0],
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:            api.MaintenanceManifestStatePending,
					MaintenanceSetID: "0",
					RunBefore:        600,
					RunAfter:         0,
					Priority:         2,
				},
			},
				&api.MaintenanceManifestDocument{
					ID:                manifestIDs[1],
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:            api.MaintenanceManifestStatePending,
						MaintenanceSetID: "1",
						RunBefore:        600,
						RunAfter:         0,
						Priority:         1,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:                manifestIDs[2],
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:            api.MaintenanceManifestStatePending,
						MaintenanceSetID: "2",
						RunBefore:        600,
						RunAfter:         1,
						Priority:         0,
					},
				})

			checker.Clear()
			checker.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
				ID:                manifestIDs[0],
				ClusterResourceID: strings.ToLower(clusterResourceID),
				MaintenanceManifest: api.MaintenanceManifest{
					State:            api.MaintenanceManifestStateCompleted,
					MaintenanceSetID: "0",
					StatusText:       "done",
					RunBefore:        600,
					RunAfter:         0,
					Priority:         2,
				},
			},
				&api.MaintenanceManifestDocument{
					ID:                manifestIDs[1],
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:            api.MaintenanceManifestStateCompleted,
						MaintenanceSetID: "1",
						StatusText:       "done",
						RunBefore:        600,
						RunAfter:         0,
						Priority:         1,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:                manifestIDs[2],
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:            api.MaintenanceManifestStateCompleted,
						MaintenanceSetID: "2",
						StatusText:       "done",
						RunBefore:        600,
						RunAfter:         1,
						Priority:         0,
					},
				})
		})

		It("runs them", func() {
			ordering := []string{}

			a.AddMaintenanceSets(map[string]sets.MaintenanceSet{
				"0": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
					ordering = append(ordering, "0")
					return api.MaintenanceManifestStateCompleted, "done"
				},
				"1": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
					ordering = append(ordering, "1")
					return api.MaintenanceManifestStateCompleted, "done"
				},
				"2": func(th mimo.TaskContext, mmd *api.MaintenanceManifestDocument, oscd *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
					ordering = append(ordering, "2")
					return api.MaintenanceManifestStateCompleted, "done"
				},
			})

			didWork, err := a.Process(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(didWork).To(BeTrue())

			// We expect 1 (start time of 0, but higher priority), then 0 (start
			// time of 0, lower priority), then 2 (start time of 1, then highest
			// priority)
			Expect(ordering).To(BeEquivalentTo([]string{"1", "0", "2"}))

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})
	})

})

func TestActuator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Actuator Suite")
}
