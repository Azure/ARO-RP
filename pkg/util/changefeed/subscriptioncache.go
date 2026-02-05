package changefeed

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"

	"github.com/Azure/ARO-RP/pkg/api"
)

// subscriptionInfo stores TenantID for a given subscription. We don't store the
// state as we filter out unwanted states in the changefeed.
type subscriptionInfo struct {
	State    api.SubscriptionState
	TenantID string
}

type SubscriptionsCache interface {
	ChangefeedResponder[*api.SubscriptionDocument]

	GetCacheSize() int
	GetSubscription(string) (*subscriptionInfo, bool)
	GetLastProcessed() (time.Time, bool)
	WaitForInitialPopulation() *sync.WaitGroup
}

func NewSubscriptionsChangefeedCache(onlyValidSubscriptions bool) *subscriptionsChangeFeedResponder {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	return &subscriptionsChangeFeedResponder{
		onlyValidSubscriptions:     onlyValidSubscriptions,
		subs:                       xsync.NewMap[string, *subscriptionInfo](),
		initialPopulationWaitGroup: wg,
	}
}

type subscriptionsChangeFeedResponder struct {
	// Do we want to only include valid (i.e. not suspended) subscriptions?
	onlyValidSubscriptions bool

	lastChangefeedProcessed    atomic.Value // time.Time
	initialPopulationWaitGroup *sync.WaitGroup

	subs *xsync.Map[string, *subscriptionInfo]
}

func (c *subscriptionsChangeFeedResponder) WaitForInitialPopulation() *sync.WaitGroup {
	return c.initialPopulationWaitGroup
}

func (c *subscriptionsChangeFeedResponder) GetSubscription(id string) (*subscriptionInfo, bool) {
	s, ok := c.subs.Load(id)
	return s, ok
}

func (c *subscriptionsChangeFeedResponder) GetCacheSize() int {
	return c.subs.Size()
}

func (c *subscriptionsChangeFeedResponder) GetLastProcessed() (time.Time, bool) {
	t, ok := c.lastChangefeedProcessed.Load().(time.Time)
	return t, ok
}

// we don't use a mutex, we use a xsync.Map
func (c *subscriptionsChangeFeedResponder) Lock()   {}
func (c *subscriptionsChangeFeedResponder) Unlock() {}

// Populate the cache with the new documents from the changefeed
func (r *subscriptionsChangeFeedResponder) OnDoc(sub *api.SubscriptionDocument) {
	id := strings.ToLower(sub.ID)

	// Don't keep subscriptions that are being deleted from our db
	if sub.Subscription.State == api.SubscriptionStateDeleted ||
		// Filter out restricted/warned subscriptions, if set
		((sub.Subscription.State == api.SubscriptionStateSuspended ||
			sub.Subscription.State == api.SubscriptionStateWarned) && r.onlyValidSubscriptions) {
		// delete is a no-op if it doesn't exist
		r.subs.Delete(id)
		return
	}

	r.subs.Compute(id, func(oldValue *subscriptionInfo, loaded bool) (*subscriptionInfo, xsync.ComputeOp) {
		TenantID := strings.ToLower(sub.Subscription.Properties.TenantID)

		// if it's the same, don't update the map
		if loaded && (oldValue.TenantID == TenantID && oldValue.State == sub.Subscription.State) {
			return nil, xsync.CancelOp
		}

		return &subscriptionInfo{
			TenantID: strings.ToLower(sub.Subscription.Properties.TenantID),
		}, xsync.UpdateOp
	})
}

func (c *subscriptionsChangeFeedResponder) OnAllPendingProcessed() {
	old := c.lastChangefeedProcessed.Swap(time.Now())
	// we've done one rotation, unlock the waitgroup
	if old == nil {
		c.initialPopulationWaitGroup.Done()
	}
}
