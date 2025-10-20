package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_proxy "github.com/Azure/ARO-RP/pkg/util/mocks/proxy"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/testliveconfig"
)

func localCosmosNewClient(_env env.Core, m metrics.Emitter, aead encryption.AEAD) (cosmosdb.DatabaseClient, error) {
	logrusEntry := _env.LoggerForComponent("database")

	masterKey := "C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw==" // To-Do: move this outside the code
	dbAuthorizer, err := cosmosdb.NewMasterKeyAuthorizer(masterKey)
	if err != nil {
		return nil, err
	}

	h, err := database.NewJSONHandle(aead)
	if err != nil {
		return nil, err
	}

	// Create HTTP client with custom transport
	c := &http.Client{
		Transport: &http.Transport{
			// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
			TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
			MaxIdleConnsPerHost: 20,
			// Skip TLS verification for local emulator with self-signed cert
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}

	databaseHostname := "127.0.0.1:8081"

	return cosmosdb.NewDatabaseClient(logrusEntry, c, h, databaseHostname, dbAuthorizer), nil
}

func createTestEnvironmentWithLocalCosmos(t *testing.T) *TestEnvironment {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	testlogger := logrus.NewEntry(logrus.StandardLogger())
	testlogger.Logger.SetLevel(logrus.DebugLevel)

	dialer := mock_proxy.NewMockDialer(ctrl)
	mockEnv := mock_env.NewMockInterface(ctrl)
	mockEnv.EXPECT().LiveConfig().Return(testliveconfig.NewTestLiveConfig(false, false)).AnyTimes()
	mockEnv.EXPECT().LoggerForComponent(gomock.Any()).Return(testlogger).AnyTimes()

	dbName := "local-unit-test-database"
	noopMetricsEmitter := noop.Noop{}
	noopClusterMetricsEmitter := noop.Noop{}

	// No encryption needed for local testing
	var aead encryption.AEAD = nil

	// Create real CosmosDB client pointing to local emulator
	localCosmosClient, err := localCosmosNewClient(mockEnv, &noopMetricsEmitter, aead)
	if err != nil {
		t.Fatalf("Failed to create local Cosmos client: %v", err)
	}

	// Create the database in CosmosDB
	localCosmosDB, err := localCosmosClient.Create(ctx, &cosmosdb.Database{ID: dbName})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create collections for each entity type
	collectionClient := cosmosdb.NewCollectionClient(localCosmosClient, dbName)

	_, err = collectionClient.Create(ctx, &cosmosdb.Collection{
		ID: "Monitors",
		PartitionKey: &cosmosdb.PartitionKey{
			Paths: []string{"/id"},
			Kind:  cosmosdb.PartitionKeyKindHash,
		},
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		t.Fatalf("Failed to create Monitors collection: %v", err)
	}

	_, err = collectionClient.Create(ctx, &cosmosdb.Collection{
		ID: "OpenShiftClusters",
		PartitionKey: &cosmosdb.PartitionKey{
			Paths: []string{"/partitionKey"},
			Kind:  cosmosdb.PartitionKeyKindHash,
		},
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		t.Fatalf("Failed to create OpenShiftClusters collection: %v", err)
	}

	_, err = collectionClient.Create(ctx, &cosmosdb.Collection{
		ID: "Subscriptions",
		PartitionKey: &cosmosdb.PartitionKey{
			Paths: []string{"/id"},
			Kind:  cosmosdb.PartitionKeyKindHash,
		},
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		t.Fatalf("Failed to create Subscriptions collection: %v", err)
	}

	// Create ALL databases using the real local Cosmos client
	monitorsDB, err := database.NewMonitors(ctx, localCosmosClient, dbName)
	if err != nil {
		t.Fatalf("Failed to create monitors DB: %v", err)
	}

	openShiftClusterDB, err := database.NewOpenShiftClusters(ctx, localCosmosClient, dbName)
	if err != nil {
		t.Fatalf("Failed to create OpenShift clusters DB: %v", err)
	}

	subscriptionsDB, err := database.NewSubscriptions(ctx, localCosmosClient, dbName)
	if err != nil {
		t.Fatalf("Failed to create subscriptions DB: %v", err)
	}

	// Create database group
	dbs := database.NewDBGroup().
		WithMonitors(monitorsDB).
		WithOpenShiftClusters(openShiftClusterDB).
		WithSubscriptions(subscriptionsDB)

	// Create master monitor document
	// _, err = monitorsDB.Create(ctx, &api.MonitorDocument{
	// 	ID: "master",
	// 	Monitor: &api.Monitor{
	// 		Buckets: make([]string, 256),
	// 	},
	// })
	// if err != nil {
	// 	t.Fatalf("Failed to create master monitor document: %v", err)
	// }

	// Initialize database fixtures
	f := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClusterDB)
	err = f.Create()
	if err != nil {
		t.Fatalf("Failed to create fixtures: %v", err)
	}

	return &TestEnvironment{
		OpenShiftClusterDB:   openShiftClusterDB,
		SubscriptionsDB:      subscriptionsDB,
		MonitorsDB:           monitorsDB,
		FakeMonitorsDBClient: nil, // Not using fake client
		Controller:           ctrl,
		TestLogger:           testlogger,
		Dialer:               dialer,
		MockEnv:              mockEnv,
		NoopMetricsEmitter:   noopMetricsEmitter,
		NoopClusterMetrics:   noopClusterMetricsEmitter,
		DBGroup:              dbs,
		localCosmosClient:    localCosmosClient,
		localCosmosDB:        localCosmosDB,
	}
}

func (env *TestEnvironment) LocalCosmosCleanup() error {
	ctx := context.Background()
	// Clean up: drop the database
	// This ensures tests start with a clean slate every time
	err := env.localCosmosClient.Delete(ctx, env.localCosmosDB)
	if err != nil {
		return err
	}

	env.Controller.Finish()
	return err
}
