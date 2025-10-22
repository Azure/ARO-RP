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

func TestClusterOperationFlow(t *testing.T) {
	// Setup test environment
	ctx := context.Background()
	env := createTestEnvironmentWithLocalCosmos(t)
	defer env.LocalCosmosCleanup()

	// Create single monitor for changefeed testing
	mon := env.CreateTestMonitor("changefeed")

	// Start changefeed
	ctxChangeFeed, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stopChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-ctxChangeFeed.Done()
		close(stopChan)
	}()

	mon.changefeedInterval = time.Second / 2
	go func() {
		mon.changefeed(ctxChangeFeed, mon.baseLog.WithField("component", "changefeed"), stopChan)
		wg.Done()
	}()

	// Create an initial subscription and cluster
	subDoc := newFakeSubscription()
	_, err := env.SubscriptionsDB.Create(ctx, subDoc)
	if err != nil {
		t.Fatalf("Couldn't create subscription in cosmos: %v", err)
	}

	clusterDoc := newFakeCluster(subDoc.ResourceID)
	clusterDoc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateCreating

	generatedCluster, err := env.OpenShiftClusterDB.Create(ctx, clusterDoc)
	if err != nil {
		t.Fatalf("Couldn't create cluster in cosmos: %v", err)
	}

	// we'll go over these steps and evaluate if the cache is being populated or not
	type lifecycleStep struct {
		name              string
		clusterState      api.ProvisioningState
		expectDocsInCache int
		expectSubsInCache int
	}

	steps := []lifecycleStep{
		{
			name:              "cluster in Creating state - should NOT be in cache",
			clusterState:      api.ProvisioningStateCreating,
			expectDocsInCache: 0,
			expectSubsInCache: 1,
		},
		{
			name:              "cluster transitions to Succeeded - should appear in cache",
			clusterState:      api.ProvisioningStateSucceeded,
			expectDocsInCache: 1,
			expectSubsInCache: 1,
		},
		{
			name:              "cluster transitions to Deleting - should disappear from cache",
			clusterState:      api.ProvisioningStateDeleting,
			expectDocsInCache: 0,
			expectSubsInCache: 1,
		},
	}

	// Execute lifecycle steps
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			// Update cluster to the new state (skip for first step since we already created it)
			if step.clusterState != api.ProvisioningStateCreating {
				generatedCluster.OpenShiftCluster.Properties.ProvisioningState = step.clusterState
				generatedCluster, err = env.OpenShiftClusterDB.Update(ctx, generatedCluster)
				if err != nil {
					t.Fatalf("Couldn't update cluster in cosmos: %v", err)
				}
			}

			// Wait for changefeed to process
			time.Sleep(time.Second)

			// Validate expected results
			if len(mon.docs) != step.expectDocsInCache {
				t.Errorf("expected %d clusters in cache, got %d", step.expectDocsInCache, len(mon.docs))
			}
			if len(mon.subs) != step.expectSubsInCache {
				t.Errorf("expected %d subscriptions in cache, got %d", step.expectSubsInCache, len(mon.subs))
			}
		})
	}
}

func TestSubscriptionFlow(t *testing.T) {
	// Setup test environment
	ctx := context.Background()
	env := createTestEnvironmentWithLocalCosmos(t)
	defer env.LocalCosmosCleanup()

	// Create single monitor for changefeed testing
	mon := env.CreateTestMonitor("changefeed")

	// Start changefeed
	ctxChangeFeed, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stopChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-ctxChangeFeed.Done()
		close(stopChan)
	}()

	mon.changefeedInterval = time.Second / 2
	go func() {
		mon.changefeed(ctxChangeFeed, mon.baseLog.WithField("component", "changefeed"), stopChan)
		wg.Done()
	}()

	// Create initial subscription
	subDoc := newFakeSubscription()
	subDoc.Subscription.State = api.SubscriptionStateRegistered

	generatedSub, err := env.SubscriptionsDB.Create(ctx, subDoc)
	if err != nil {
		t.Fatalf("Couldn't create subscription in cosmos: %v", err)
	}

	// Define subscription lifecycle steps
	type lifecycleStep struct {
		name              string
		subscriptionState api.SubscriptionState
		expectSubsInCache int
	}

	steps := []lifecycleStep{
		{
			name:              "subscription in Registered state - should be in cache",
			subscriptionState: api.SubscriptionStateRegistered,
			expectSubsInCache: 1,
		},
		{
			name:              "subscription transitions to Deleted - should disappear from cache",
			subscriptionState: api.SubscriptionStateDeleted,
			expectSubsInCache: 0,
		},
	}

	// Execute lifecycle steps
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			// Update subscription to the new state (skip for first step since we already created it)
			if step.subscriptionState != api.SubscriptionStateRegistered {
				generatedSub.Subscription.State = step.subscriptionState
				generatedSub, err = env.SubscriptionsDB.Update(ctx, generatedSub)
				if err != nil {
					t.Fatalf("Couldn't update subscription in cosmos: %v", err)
				}
			}

			// Wait for changefeed to process
			time.Sleep(time.Second)

			// Validate expected results
			if len(mon.subs) != step.expectSubsInCache {
				t.Errorf("expected %d subscriptions in cache, got %d", step.expectSubsInCache, len(mon.subs))
			}
		})
	}
}
