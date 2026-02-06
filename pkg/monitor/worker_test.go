package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	testlog "github.com/Azure/ARO-RP/test/util/log"
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

// closeableMonitor implements both Monitor and Closeable interfaces for testing
type closeableMonitor struct {
	closeCalled bool
	mu          sync.Mutex
}

func (m *closeableMonitor) Monitor(ctx context.Context) error {
	return nil
}

func (m *closeableMonitor) MonitorName() string {
	return "closeableMonitor"
}

func (m *closeableMonitor) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
}

func (m *closeableMonitor) wasClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

// nonCloseableMonitor only implements Monitor, not Closeable
type nonCloseableMonitor struct{}

func (m *nonCloseableMonitor) Monitor(ctx context.Context) error {
	return nil
}

func (m *nonCloseableMonitor) MonitorName() string {
	return "nonCloseableMonitor"
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
	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Create single monitor for changefeed testing
	mon := env.CreateTestMonitor("changefeed")

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
		// Running changefeed loop every second
		mon.changefeed(ctx, mon.baseLog.WithField("component", "changefeed"), stopChan)
		wg.Done()
	}()

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
		}, {
			name:                          "create cluster in creating state - should be ignored",
			action:                        "create",
			clusterProvisioningState:      api.ProvisioningStateCreating,
			subscriptionProvisioningState: api.SubscriptionStateRegistered,
			expectDocs:                    2,
			expectSubs:                    4,
		}, {
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

			// Wait for changefeed to process
			time.Sleep(2 * time.Second)

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

func TestCloseMonitors(t *testing.T) {
	for _, tt := range []struct {
		name     string
		monitors []monitoring.Monitor
		expected []bool // whether each closeable monitor should have Close() called
	}{
		{
			name:     "empty monitors list",
			monitors: []monitoring.Monitor{},
			expected: []bool{},
		},
		{
			name: "only closeable monitors",
			monitors: []monitoring.Monitor{
				&closeableMonitor{},
				&closeableMonitor{},
			},
			expected: []bool{true, true},
		},
		{
			name: "only non-closeable monitors",
			monitors: []monitoring.Monitor{
				&nonCloseableMonitor{},
				&nonCloseableMonitor{},
			},
			expected: []bool{},
		},
		{
			name: "mixed closeable and non-closeable monitors",
			monitors: []monitoring.Monitor{
				&closeableMonitor{},
				&nonCloseableMonitor{},
				&closeableMonitor{},
			},
			expected: []bool{true, true},
		},
		{
			name:     "nil monitors list (should not panic)",
			monitors: nil,
			expected: []bool{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Call closeMonitors
			closeMonitors(tt.monitors)

			// Verify closeable monitors had Close() called
			closeableIdx := 0
			for _, m := range tt.monitors {
				if cm, ok := m.(*closeableMonitor); ok {
					if closeableIdx < len(tt.expected) {
						if cm.wasClosed() != tt.expected[closeableIdx] {
							t.Errorf("closeable monitor %d: expected Close() called = %v, got %v",
								closeableIdx, tt.expected[closeableIdx], cm.wasClosed())
						}
						closeableIdx++
					}
				}
			}
		})
	}
}
