package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_proxy "github.com/Azure/ARO-RP/pkg/util/mocks/proxy"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	"github.com/Azure/ARO-RP/test/util/testliveconfig"
)

type panicMonitor struct {
}

func (m *panicMonitor) Monitor(ctx context.Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = &monitoring.MonitorPanic{PanicValue: e}
		}
	}()
	panic("oh no!")
}

func (m *panicMonitor) MonitorName() string {
	return "panicMonitor"
}

func TestExecute(t *testing.T) {
	_, log := testlog.New()
	pm := &panicMonitor{}

	triggeredFail := false
	onPanic := func(m monitoring.Monitor) {
		fmt.Println("failed")
		triggeredFail = true
	}

	allJobsDone := make(chan bool)
	go execute(context.Background(), log, allJobsDone, []monitoring.Monitor{pm}, onPanic)

	<-allJobsDone
	assert.True(t, triggeredFail)
}

func TestChangefeedOperations(t *testing.T) {
	// Setup single monitor for all test operations
	openShiftClusterDB, _ := testdatabase.NewFakeOpenShiftClusters()
	subscriptionsDB, _ := testdatabase.NewFakeSubscriptions()
	monitorsDB, fakeMonitorsDBClient := testdatabase.NewFakeMonitors()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testlogger := logrus.NewEntry(logrus.StandardLogger())
	testlogger.Logger.SetLevel(logrus.DebugLevel)
	dialer := mock_proxy.NewMockDialer(ctrl)
	mockEnv := mock_env.NewMockInterface(ctrl)
	mockEnv.EXPECT().LiveConfig().Return(testliveconfig.NewTestLiveConfig(false, false)).AnyTimes()
	noopMetricsEmitter := noop.Noop{}
	noopClusterMetricsEmitter := noop.Noop{}
	dbs := database.NewDBGroup().
		WithMonitors(testdatabase.NewFakeMonitorWithExistingClient(fakeMonitorsDBClient)).
		WithOpenShiftClusters(openShiftClusterDB).
		WithSubscriptions(subscriptionsDB)

	mon := NewMonitor(testlogger, dialer, dbs, &noopMetricsEmitter, &noopClusterMetricsEmitter, mockEnv).(*monitor)
	mon.nsgMonitorBuilder = fakeNsgMonitoringBuilder
	mon.hiveMonitorBuilder = fakeHiveMonitoringBuilder
	mon.clusterMonitorBuilder = fakeClusterMonitorBuilder

	monitorsDB.Create(context.TODO(), &api.MonitorDocument{
		ID: "master",
		Monitor: &api.Monitor{
			Buckets: make([]string, 256),
		},
	})

	f := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClusterDB)
	f.Create()

	// Start changefeed
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stopChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		close(stopChan)
	}()

	mon.changefeedInterval = time.Second / 2
	go func() {
		// Running changefeed loop every half second
		mon.changefeed(ctx, mon.baseLog.WithField("component", "changefeed"), stopChan)
		wg.Done()
	}()

	type operation struct {
		name              string
		action            string // "create"
		provisioningState api.ProvisioningState
		expectDocs        int
		expectSubs        int
	}

	operations := []operation{
		{
			name:              "create first cluster with subscription",
			action:            "create",
			provisioningState: api.ProvisioningStateSucceeded,
			expectDocs:        1,
			expectSubs:        1,
		},
		{
			name:              "create second cluster with new subscription",
			action:            "create",
			provisioningState: api.ProvisioningStateSucceeded,
			expectDocs:        2,
			expectSubs:        2,
		},
		{
			name:              "create cluster in Deleting state - should be ignored",
			action:            "create",
			provisioningState: api.ProvisioningStateDeleting,
			expectDocs:        2,
			expectSubs:        2,
		},
	}

	// Execute operations in sequence
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			// Create subscription and cluster documents
			subDoc := newFakeSubscription()
			clusterDoc := newFakeCluster(subDoc.ResourceID)
			clusterDoc.OpenShiftCluster.Properties.ProvisioningState = op.provisioningState

			switch op.action {
			case "create":
				_, err := openShiftClusterDB.Create(context.Background(), clusterDoc)
				if err != nil {
					t.Fatalf("Couldn't create cluster doc: %v", err)
				}
				_, err = subscriptionsDB.Create(context.Background(), subDoc)
				if err != nil {
					t.Fatalf("Couldn't create subscription doc: %v", err)
				}

			}

			// Wait for changefeed to process
			time.Sleep(time.Second)

			// Validate expected results
			if len(mon.docs) != op.expectDocs {
				t.Errorf("%s: expected %d documents in cache, got %d", op.name, op.expectDocs, len(mon.docs))
			}
			if len(mon.subs) != op.expectSubs {
				t.Errorf("%s: expected %d subscriptions in cache, got %d", op.name, op.expectSubs, len(mon.subs))
			}
		})
	}
}
