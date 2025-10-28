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
	testlocalcosmos "github.com/Azure/ARO-RP/test/localcosmos"
)

var _ = Describe("MIMO Actuator Service (CosmosDB Emulator)", Ordered, func() {
	var fixtures *testdatabase.Fixture
	var checker *testdatabase.Checker
	var dbConn cosmosdb.DatabaseClient
	var manifests database.MaintenanceManifests
	var manifestsClient cosmosdb.MaintenanceManifestDocumentClient
	var clusters database.OpenShiftClusters
	var m *fakeMetricsEmitter

	var cosmosDBEmulator testlocalcosmos.LocalCosmosDB

	var svc *service

	var ctx context.Context
	var cancel context.CancelFunc

	var log *logrus.Entry
	var _env env.Interface

	var controller *gomock.Controller

	mockSubID := "00000000-0000-0000-0000-000000000000"
	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	AfterAll(func() {
		// err := cosmosDBEmulator.Stop()
		// if err != nil {
		// 	GinkgoWriter.Printf("Error stopping CosmosDB emulator: %v\n", err)
		// }

		if cancel != nil {
			cancel()
		}

		if controller != nil {
			controller.Finish()
		}
	})

	BeforeAll(func() {
		log = logrus.NewEntry(&logrus.Logger{
			Out:       GinkgoWriter,
			Formatter: new(logrus.TextFormatter),
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.DebugLevel,
		})

		controller = gomock.NewController(nil)
		_e := mock_env.NewMockInterface(controller)
		_e.EXPECT().LoggerForComponent(gomock.Any()).Return(log).AnyTimes()
		_env = _e

		ctx, cancel = context.WithCancel(context.Background())

		conn, err := testlocalcosmos.GetPodmanConnection(ctx)
		Expect(err).ToNot(HaveOccurred())

		cosmosDBEmulator = testlocalcosmos.NewLocalCosmos(conn)

		err = cosmosDBEmulator.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		m = newfakeMetricsEmitter()

		dbConn, err = testlocalcosmos.GetConnection(_env, m, testdatabase.NewFakeAEAD())
		Expect(err).ToNot(HaveOccurred())

		fixtures = testdatabase.NewFixture()
		checker = testdatabase.NewChecker()
	})

	BeforeEach(func() {
		id, err := testlocalcosmos.CreateFreshDB(ctx, dbConn)
		Expect(err).ToNot(HaveOccurred())

		GinkgoWriter.Printf("Using CosmosDB database %s\n", id)

		// Want to have a method that's not a part of the interface, but is
		// usable in tests? Just don't declare it as a part of the interface
		// until you've used the methods on the struct but not on the interface!
		cl, err := database.NewOpenShiftClusters(ctx, dbConn, id)
		Expect(err).ToNot(HaveOccurred())
		clusters = cl

		manifestsObject, err := database.NewMaintenanceManifests(ctx, dbConn, id)
		Expect(err).ToNot(HaveOccurred())
		manifestsClient = manifestsObject.Client()
		manifests = manifestsObject

		now := func() time.Time { return time.Unix(120, 0) }
		dbg := database.NewDBGroup().WithMaintenanceManifests(manifests).WithOpenShiftClusters(clusters)

		svc = NewService(_env, log, nil, dbg, m, []int{1})
		svc.now = now
		svc.workerDelay = func() time.Duration { return 0 * time.Second }
		svc.serveHealthz = false
	})

	JustBeforeEach(func() {
		m.Clear()

		err := fixtures.WithOpenShiftClusters(clusters).WithMaintenanceManifests(manifests).Create()
		Expect(err).ToNot(HaveOccurred())
	})

	When("clusters are polled", func() {
		BeforeEach(func() {
			fixtures.Clear()
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key:    strings.ToLower(clusterResourceID),
				Bucket: 1,

				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
					},
				},
			})
			fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key:    strings.ToLower(clusterResourceID + "ignored"),
				Bucket: 2,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID + "ignored",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
					},
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

			// Contains one that we check and one that we don't
			Expect(newOld).To(HaveLen(2))
		})

		It("removes clusters if they are not in the doc", func() {
			svc.b.UpsertDoc(&api.OpenShiftClusterDocument{Key: clusterResourceID + "2"})

			lastGotDocs := make(map[string]*api.OpenShiftClusterDocument)
			lastGotDocs[clusterResourceID+"2"] = &api.OpenShiftClusterDocument{Key: clusterResourceID + "2"}

			newOld, err := svc.poll(ctx, lastGotDocs)
			Expect(err).ToNot(HaveOccurred())

			// Contains one that we check and one that we don't
			Expect(newOld).To(HaveLen(2))
		})
	})

	When("maintenance needs to occur", func() {
		var manifestID string

		BeforeEach(func() {
			fixtures.Clear()
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

			futureTime := int(time.Now().Add(60 * time.Second).Unix())

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
						RunAfter:  1,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:                manifestID2,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						RunBefore:         futureTime,
						RunAfter:          1,
						MaintenanceTaskID: "0000-0000-0001",
					},
				},
				// A manifest for a cluster that is not served by our bucket allocation
				&api.MaintenanceManifestDocument{
					ID:                manifestID3,
					ClusterResourceID: strings.ToLower(clusterResourceID + "ignored"),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						RunBefore:         futureTime,
						RunAfter:          1,
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
						RunAfter:   1,
					},
				},
				&api.MaintenanceManifestDocument{
					ID:                manifestID2,
					Dequeues:          1,
					ClusterResourceID: strings.ToLower(clusterResourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStateCompleted,
						StatusText:        "ok",
						RunBefore:         futureTime,
						RunAfter:          1,
						MaintenanceTaskID: "0000-0000-0001",
					},
				},
				// manifest will not be served
				&api.MaintenanceManifestDocument{
					ID:                manifestID3,
					ClusterResourceID: strings.ToLower(clusterResourceID + "ignored"),
					MaintenanceManifest: api.MaintenanceManifest{
						State:             api.MaintenanceManifestStatePending,
						RunBefore:         futureTime,
						RunAfter:          1,
						MaintenanceTaskID: "0000-0000-0001",
					},
				},
			)
		})

		It("expires them", func() {
			svc.pollTime = time.Second

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
			svc.workerRoutines.Wait()

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})

		It("loads the full cluster document", func() {
			svc.pollTime = time.Second

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
			svc.workerRoutines.Wait()

			errs := checker.CheckMaintenanceManifests(manifestsClient)
			Expect(errs).To(BeNil(), fmt.Sprintf("%v", errs))
		})
	})
})
