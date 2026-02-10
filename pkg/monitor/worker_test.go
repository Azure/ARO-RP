package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
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
		triggeredFail = true
	}

	allJobsDone := make(chan bool)
	go execute(context.Background(), log, allJobsDone, []monitoring.Monitor{pm}, onPanic)

	<-allJobsDone
	assert.True(t, triggeredFail)
}

func TestChangefeedOperations(t *testing.T) {
	// Previous version of the tests
	// Setup test environment using the old fake client (to maintain these checks in the CI)
	env := SetupTestEnvironmentWithFakeClient(t)
	defer env.Cleanup()

	// Create single monitor for changefeed testing
	mon := env.CreateTestMonitor("changefeed")

	// Start changefeed
	ctxChangeFeed, cancel := context.WithTimeout(env.ctx, 20*time.Second)
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
		// Running changefeed loop every second
		mon.changefeed(ctxChangeFeed, mon.baseLog.WithField("component", "changefeed"), stopChan)
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
				_, err := env.OpenShiftClusterDB.Create(env.ctx, clusterDoc)
				if err != nil {
					t.Fatalf("Couldn't create cluster doc: %v", err)
				}
				_, err = env.SubscriptionsDB.Create(env.ctx, subDoc)
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

func TestClusterOperationFlow(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.LocalCosmosCleanup()
	if env.localCosmosClient == nil && env.localCosmosDB == nil {
		return // this only works with a local cosmosdb
	}

	mon := env.CreateTestMonitor("changefeed")

	ctxChangeFeed, cancel := context.WithTimeout(env.ctx, 20*time.Second)
	defer cancel()

	stopChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-ctxChangeFeed.Done()
		close(stopChan)
	}()

	mon.changefeedInterval = time.Second / 2
	go func() { // We need a changefeed goroutine to retrieve changes in CosmosDB
		mon.changefeed(ctxChangeFeed, mon.baseLog.WithField("component", "changefeed"), stopChan)
		wg.Done()
	}()

	subDoc := newFakeSubscription()
	_, err := env.SubscriptionsDB.Create(env.ctx, subDoc)
	if err != nil {
		t.Fatalf("Couldn't create subscription in cosmos: %v", err)
	}

	clusterDoc := newFakeCluster(subDoc.ResourceID)
	clusterDoc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateCreating

	generatedCluster, err := env.OpenShiftClusterDB.Create(env.ctx, clusterDoc)
	if err != nil {
		t.Fatalf("Couldn't create cluster in cosmos: %v", err)
	}

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

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			// Update cluster to the new state (skip for first step since we already created it)
			if step.clusterState != api.ProvisioningStateCreating {
				generatedCluster.OpenShiftCluster.Properties.ProvisioningState = step.clusterState
				generatedCluster, err = env.OpenShiftClusterDB.Update(env.ctx, generatedCluster)
				if err != nil {
					t.Fatalf("Couldn't update cluster in cosmos: %v", err)
				}
			}

			// Wait for changefeed to process
			time.Sleep(time.Second)

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
	env := SetupTestEnvironment(t)
	defer env.LocalCosmosCleanup()
	if env.localCosmosClient == nil && env.localCosmosDB == nil {
		return // this only works with a local cosmosdb
	}

	mon := env.CreateTestMonitor("changefeed")

	ctxChangeFeed, cancel := context.WithTimeout(env.ctx, 20*time.Second)
	defer cancel()

	stopChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-ctxChangeFeed.Done()
		close(stopChan)
	}()

	mon.changefeedInterval = time.Second / 2
	go func() { // We need a changefeed goroutine to retrieve changes in CosmosDB
		mon.changefeed(ctxChangeFeed, mon.baseLog.WithField("component", "changefeed"), stopChan)
		wg.Done()
	}()

	subDoc := newFakeSubscription()
	subDoc.Subscription.State = api.SubscriptionStateRegistered

	generatedSub, err := env.SubscriptionsDB.Create(env.ctx, subDoc)
	if err != nil {
		t.Fatalf("Couldn't create subscription in cosmos: %v", err)
	}

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

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			// Update subscription to the new state (skip for first step since we already created it)
			if step.subscriptionState != api.SubscriptionStateRegistered {
				generatedSub.Subscription.State = step.subscriptionState
				generatedSub, err = env.SubscriptionsDB.Update(env.ctx, generatedSub)
				if err != nil {
					t.Fatalf("Couldn't update subscription in cosmos: %v", err)
				}
			}

			// Wait for changefeed to process
			time.Sleep(time.Second)

			if len(mon.subs) != step.expectSubsInCache {
				t.Errorf("expected %d subscriptions in cache, got %d", step.expectSubsInCache, len(mon.subs))
			}
		})
	}
}
