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

// subscriptionInfo is used as the value in a map with subscriptionID as the
// key. It holds the TenantID and state for the given subscription, as the rest
// of the data in the document is superfluous and can be quite large due to
// MissingFields
type subscriptionInfo struct {
	State    api.SubscriptionState
	TenantID string
}

type SubscriptionsCache interface {
	ChangefeedConsumer[*api.SubscriptionDocument]

	GetCacheSize() int
	GetSubscription(string) (subscriptionInfo, bool)
	GetLastProcessed() (time.Time, bool)
	WaitForInitialPopulation()
}

func NewSubscriptionsChangefeedCache(onlyValidSubscriptions bool) *subscriptionsChangeFeedResponder {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	return &subscriptionsChangeFeedResponder{
		onlyValidSubscriptions:     onlyValidSubscriptions,
		subs:                       xsync.NewMap[string, subscriptionInfo](),
		initialPopulationWaitGroup: wg,
	}
}

type subscriptionsChangeFeedResponder struct {
	// Do we want to only include valid (i.e. not suspended) subscriptions?
	onlyValidSubscriptions bool

	lastChangefeedProcessed    atomic.Value // time.Time
	initialPopulationWaitGroup *sync.WaitGroup

	subs *xsync.Map[string, subscriptionInfo]
}

var _ SubscriptionsCache = &subscriptionsChangeFeedResponder{}

func (c *subscriptionsChangeFeedResponder) WaitForInitialPopulation() {
	c.initialPopulationWaitGroup.Wait()
}

func (c *subscriptionsChangeFeedResponder) GetSubscription(id string) (subscriptionInfo, bool) {
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

// we don't use a mutex internally, we use a xsync.Map, so Lock/Unlock are
// no-ops
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

	r.subs.Compute(id, func(oldValue subscriptionInfo, loaded bool) (subscriptionInfo, xsync.ComputeOp) {
		new := subscriptionInfo{
			TenantID: strings.ToLower(sub.Subscription.Properties.TenantID),
			State:    sub.Subscription.State,
		}

		// if it's the same, don't update the map
		if oldValue == new {
			return new, xsync.CancelOp
		}
		return new, xsync.UpdateOp
	})
}

func (c *subscriptionsChangeFeedResponder) OnAllPendingProcessed() {
	old := c.lastChangefeedProcessed.Swap(time.Now())
	// we've consumed the initial documents, unlock the waitgroup
	if old == nil {
		c.initialPopulationWaitGroup.Done()
	}
}
