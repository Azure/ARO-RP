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

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	pkgenv "github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type panicMonitor struct{}

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

type slowMonitor struct{}

func (m *slowMonitor) Monitor(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (m *slowMonitor) MonitorName() string {
	return "slowMonitor"
}

type closeableMonitor struct {
	monitorFn  func(context.Context) error
	closeOnce  sync.Once
	closedChan chan struct{}
	doneChan   chan struct{}
}

func newCloseableMonitor(monitorFn func(context.Context) error) *closeableMonitor {
	return &closeableMonitor{
		monitorFn:  monitorFn,
		closedChan: make(chan struct{}),
		doneChan:   make(chan struct{}),
	}
}

func (m *closeableMonitor) Monitor(ctx context.Context) error {
	defer close(m.doneChan)
	return m.monitorFn(ctx)
}

func (m *closeableMonitor) MonitorName() string {
	return "closeableMonitor"
}

func (m *closeableMonitor) Close() {
	m.closeOnce.Do(func() {
		close(m.closedChan)
	})
}

func channelClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// TestExecuteReturnsWhenNoReceiver verifies that the execute goroutine does not
// leak when nobody reads from the done channel (the timeout path in workOne).
// With an unbuffered channel this test would hang forever.
func TestExecuteReturnsWhenNoReceiver(t *testing.T) {
	_, log := testlog.New()

	ctx, cancel := context.WithCancel(context.Background())
	onPanic := func(m monitoring.Monitor) {}

	// Buffered channel: execute can send without a receiver
	done := make(chan bool, 1)
	go execute(ctx, log, done, []monitoring.Monitor{&slowMonitor{}}, onPanic)

	// Simulate workOne's timeout path: cancel the context and never read from done
	cancel()

	// The execute goroutine must exit within a reasonable time
	assert.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond, "execute goroutine leaked: blocked sending on done channel")
}

func TestCloseMonitorsClosesOnlyCloseableMonitors(t *testing.T) {
	first := newCloseableMonitor(func(context.Context) error { return nil })
	second := &monitoring.NoOpMonitor{}
	third := newCloseableMonitor(func(context.Context) error { return nil })

	closeMonitors([]monitoring.Monitor{first, second, third})

	assert.True(t, channelClosed(first.closedChan))
	assert.True(t, channelClosed(third.closedChan))
}

func TestWorkOneWaitsForMonitorCompletionWithinGracePeriod(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	oldGracePeriod := monitorCleanupGracePeriod
	monitorCleanupGracePeriod = 50 * time.Millisecond
	t.Cleanup(func() {
		monitorCleanupGracePeriod = oldGracePeriod
	})

	clusterMon := newCloseableMonitor(func(ctx context.Context) error {
		<-ctx.Done()
		time.Sleep(10 * time.Millisecond)
		return ctx.Err()
	})

	mon := env.CreateTestMonitor("workone-graceful")
	mon.clusterMonitorBuilder = func(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster, _ pkgenv.Interface, tenantID string, m metrics.Emitter, hourlyRun bool) (monitoring.Monitor, error) {
		return clusterMon, nil
	}

	subDoc := newFakeSubscription()
	clusterDoc := newFakeCluster(subDoc.ResourceID)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mon.workOne(ctx, env.TestLogger, clusterDoc, subDoc.ResourceID, subDoc.Subscription.Properties.TenantID, false, ticker)

	assert.True(t, channelClosed(clusterMon.doneChan), "monitor should finish before workOne returns")
	assert.True(t, channelClosed(clusterMon.closedChan), "closeable monitor should be closed when workOne returns")
}

func TestWorkOneForcedCleanupAfterGracePeriod(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	oldGracePeriod := monitorCleanupGracePeriod
	monitorCleanupGracePeriod = 20 * time.Millisecond
	t.Cleanup(func() {
		monitorCleanupGracePeriod = oldGracePeriod
	})

	clusterMon := newCloseableMonitor(func(context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	mon := env.CreateTestMonitor("workone-forced-cleanup")
	mon.clusterMonitorBuilder = func(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster, _ pkgenv.Interface, tenantID string, m metrics.Emitter, hourlyRun bool) (monitoring.Monitor, error) {
		return clusterMon, nil
	}

	subDoc := newFakeSubscription()
	clusterDoc := newFakeCluster(subDoc.ResourceID)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	mon.workOne(ctx, env.TestLogger, clusterDoc, subDoc.ResourceID, subDoc.Subscription.Properties.TenantID, false, ticker)
	elapsed := time.Since(start)

	assert.True(t, channelClosed(clusterMon.closedChan), "closeable monitor should be closed on forced cleanup")
	assert.False(t, channelClosed(clusterMon.doneChan), "monitor should still be running when grace period expires")
	assert.Less(t, elapsed, 100*time.Millisecond, "workOne should return before the stubborn monitor finishes")
	assert.Eventually(t, func() bool {
		return channelClosed(clusterMon.doneChan)
	}, time.Second, 10*time.Millisecond, "stubborn monitor should eventually finish to avoid leaking the test goroutine")
}

func TestChangefeedOperations(t *testing.T) {
	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Create single monitor for changefeed testing
	mon := env.CreateTestMonitor("changefeed")

	// Start changefeed
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	stopChan := make(chan struct{})

	var lastClusterDataUpdate time.Time
	var lastSubDataUpdate time.Time

	mon.changefeedInterval = time.Millisecond * 5
	mon.startChangefeeds(ctx, stopChan)

	type operation struct {
		name                          string
		action                        string // "create"
		clusterProvisioningState      api.ProvisioningState
		subscriptionProvisioningState api.SubscriptionState
		expectDocs                    int
		expectSubs                    int
	}

	operations := []operation{
		{
			name:                          "create first cluster with subscription",
			action:                        "create",
			clusterProvisioningState:      api.ProvisioningStateSucceeded,
			subscriptionProvisioningState: api.SubscriptionStateRegistered,
			expectDocs:                    1,
			expectSubs:                    1,
		},
		{
			name:                          "create second cluster with new subscription",
			action:                        "create",
			clusterProvisioningState:      api.ProvisioningStateSucceeded,
			subscriptionProvisioningState: api.SubscriptionStateRegistered,
			expectDocs:                    2,
			expectSubs:                    2,
		},
		{
			name:                          "create cluster in Deleting state - should be ignored",
			action:                        "create",
			clusterProvisioningState:      api.ProvisioningStateDeleting,
			subscriptionProvisioningState: api.SubscriptionStateRegistered,
			expectDocs:                    2,
			expectSubs:                    3,
		},
		{
			name:                          "create cluster in creating state - should be ignored",
			action:                        "create",
			clusterProvisioningState:      api.ProvisioningStateCreating,
			subscriptionProvisioningState: api.SubscriptionStateRegistered,
			expectDocs:                    2,
			expectSubs:                    4,
		},
		{
			name:                          "subscription and cluster in Deleting state - BOTH should be ignored",
			action:                        "create",
			clusterProvisioningState:      api.ProvisioningStateDeleting,
			subscriptionProvisioningState: api.SubscriptionStateDeleted,
			expectDocs:                    2,
			expectSubs:                    4,
		},
	}

	// Execute operations in sequence
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			// Create subscription and cluster documents
			subDoc := newFakeSubscription()
			subDoc.Subscription.State = op.subscriptionProvisioningState
			clusterDoc := newFakeCluster(subDoc.ResourceID)
			clusterDoc.OpenShiftCluster.Properties.ProvisioningState = op.clusterProvisioningState

			switch op.action {
			case "create":
				_, err := env.OpenShiftClusterDB.Create(context.Background(), clusterDoc)
				if err != nil {
					t.Fatalf("Couldn't create cluster doc: %v", err)
				}
				_, err = env.SubscriptionsDB.Create(context.Background(), subDoc)
				if err != nil {
					t.Fatalf("Couldn't create subscription doc: %v", err)
				}
			}

			// Wait for changefeeds to be consumed
			assert.Eventually(t, func() bool {
				lastProc, _ := mon.subs.GetLastProcessed()
				lastData, _ := mon.subs.GetLastDataUpdate()
				return lastData != lastSubDataUpdate && lastProc != lastData
			}, time.Second, 1*time.Millisecond)
			assert.Eventually(t, func() bool {
				lastProc, _ := mon.clusters.GetLastProcessed()
				lastData, _ := mon.clusters.GetLastDataUpdate()
				return lastData != lastClusterDataUpdate && lastProc != lastData
			}, time.Second, 1*time.Millisecond)

			lastClusterDataUpdate, _ = mon.clusters.GetLastDataUpdate()
			lastSubDataUpdate, _ = mon.subs.GetLastDataUpdate()

			// Validate expected results
			if mon.clusters.GetCacheSize() != op.expectDocs {
				t.Errorf("%s: expected %d documents in cache, got %d", op.name, op.expectDocs, mon.clusters.GetCacheSize())
			}
			if mon.subs.GetCacheSize() != op.expectSubs {
				t.Errorf("%s: expected %d subscriptions in cache, got %d", op.name, op.expectSubs, mon.subs.GetCacheSize())
			}
		})
	}
}
