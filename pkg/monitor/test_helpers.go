package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_proxy "github.com/Azure/ARO-RP/pkg/util/mocks/proxy"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/testliveconfig"
)

// Global test variables
var fakeClusterVisitMonitoringAttempts = xsync.NewMap[string, *atomic.Int64]()

// TestEnvironment contains all the test setup components
type TestEnvironment struct {
	OpenShiftClusterDB      database.OpenShiftClusters
	SubscriptionsDB         database.Subscriptions
	PoolWorkersDB           database.PoolWorkers
	OpenShiftClusterClient  *cosmosdb.FakeOpenShiftClusterDocumentClient
	SubscriptionsClient     *cosmosdb.FakeSubscriptionDocumentClient
	FakePoolWorkersDBClient *cosmosdb.FakePoolWorkerDocumentClient
	Controller              *gomock.Controller
	TestLogger              *logrus.Entry
	Dialer                  *mock_proxy.MockDialer
	MockEnv                 *mock_env.MockInterface
	NoopMetricsEmitter      noop.Noop
	NoopClusterMetrics      noop.Noop
	DBGroup                 monitorDBs
}

// SetupTestEnvironment creates a common test environment for monitor tests
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	// Create databases
	openShiftClusterDB, openShiftClusterClient := testdatabase.NewFakeOpenShiftClusters()
	subscriptionsDB, subscriptionsClient := testdatabase.NewFakeSubscriptions()
	poolWorkersDB, fakePoolMonitorsDBClient := testdatabase.NewFakePoolWorkers(time.Now)

	// Create mocks
	ctrl := gomock.NewController(t)
	testlogger := logrus.NewEntry(logrus.StandardLogger())
	testlogger.Logger.SetLevel(logrus.DebugLevel)
	dialer := mock_proxy.NewMockDialer(ctrl)
	mockEnv := mock_env.NewMockInterface(ctrl)
	mockEnv.EXPECT().LiveConfig().Return(testliveconfig.NewTestLiveConfig(false, false)).AnyTimes()

	// Create metrics emitters
	noopMetricsEmitter := noop.Noop{}
	noopClusterMetricsEmitter := noop.Noop{}

	// Create database group
	dbs := database.NewDBGroup().
		WithPoolWorkers(poolWorkersDB).
		WithOpenShiftClusters(openShiftClusterDB).
		WithSubscriptions(subscriptionsDB)

	// Initialize database fixtures
	f := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClusterDB)
	f.Create()

	return &TestEnvironment{
		OpenShiftClusterDB:      openShiftClusterDB,
		SubscriptionsDB:         subscriptionsDB,
		PoolWorkersDB:           poolWorkersDB,
		OpenShiftClusterClient:  openShiftClusterClient,
		SubscriptionsClient:     subscriptionsClient,
		FakePoolWorkersDBClient: fakePoolMonitorsDBClient,
		Controller:              ctrl,
		TestLogger:              testlogger,
		Dialer:                  dialer,
		MockEnv:                 mockEnv,
		NoopMetricsEmitter:      noopMetricsEmitter,
		NoopClusterMetrics:      noopClusterMetricsEmitter,
		DBGroup:                 dbs,
	}
}

// CreateTestMonitor creates a single monitor with test configuration
func (env *TestEnvironment) CreateTestMonitor(loggerField string) *monitor {
	uniquePoolWorkersDB := testdatabase.NewFakePoolWorkersWithExistingClient(env.FakePoolWorkersDBClient)
	nDBs := database.NewDBGroup().WithPoolWorkers(uniquePoolWorkersDB).
		WithOpenShiftClusters(env.OpenShiftClusterDB).
		WithSubscriptions(env.SubscriptionsDB)

	mon := NewMonitor(
		env.TestLogger.WithField("test", loggerField),
		env.Dialer,
		nDBs,
		&env.NoopMetricsEmitter,
		&env.NoopClusterMetrics,
		env.MockEnv,
	).(*monitor)

	// Apply test-specific configurations
	mon.nsgMonitorBuilder = fakeNsgMonitoringBuilder
	mon.hiveMonitorBuilder = fakeHiveMonitoringBuilder
	mon.clusterMonitorBuilder = fakeClusterMonitorBuilder
	mon.workerMaxStartupDelay = 0
	mon.interval = 250 * time.Millisecond
	mon.changefeedInterval = 100 * time.Millisecond
	mon.bucketRefreshInterval = 100 * time.Millisecond
	mon.readyDelay = 250 * time.Millisecond

	return mon
}

// Cleanup performs test cleanup
func (env *TestEnvironment) Cleanup() {
	env.Controller.Finish()
}

// Fake monitoring builders for testing
func fakeClusterMonitorBuilder(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster, env env.Interface, tenantID string, m metrics.Emitter, hourlyRun bool) (monitoring.Monitor, error) {
	counter, ok := fakeClusterVisitMonitoringAttempts.Load(oc.ID)
	if !ok {
		return nil, fmt.Errorf("didn't find counter for %s", oc.ID)
	}

	return &fakeMonitor{
		timeout:        2 * time.Second,
		clusterCounter: counter,
	}, nil
}

func fakeHiveMonitoringBuilder(log *logrus.Entry, oc *api.OpenShiftCluster, m metrics.Emitter, hourlyRun bool, hiveClusterManager hive.ClusterManager) (monitoring.Monitor, error) {
	return &monitoring.NoOpMonitor{}, nil
}

func fakeNsgMonitoringBuilder(log *logrus.Entry, oc *api.OpenShiftCluster, e env.Interface, subscriptionID, tenantID string, emitter metrics.Emitter, dims map[string]string, trigger <-chan time.Time) monitoring.Monitor {
	return &monitoring.NoOpMonitor{}
}

type fakeMonitor struct {
	timeout        time.Duration
	clusterCounter *atomic.Int64
}

func (fm *fakeMonitor) Monitor(ctx context.Context) error {
	time.Sleep(fm.timeout)
	fm.clusterCounter.Add(1)
	return nil
}

func (fm *fakeMonitor) MonitorName() string {
	return "fakemonitor"
}
