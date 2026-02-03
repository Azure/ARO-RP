package changefeed

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
)

// subscriptionInfo stores TenantID for a given subscription. We don't store the
// state as we filter out unwanted states in the changefeed.
type subscriptionInfo struct {
	TenantID string
}

type SubscriptionsCache interface {
	ChangefeedResponder[*api.SubscriptionDocument]

	GetCacheSize() int
	GetSubscription(string) (*subscriptionInfo, bool)
}

func NewSubscriptionsChangefeedCache() *subscriptionsChangeFeedResponder {
	return &subscriptionsChangeFeedResponder{
		subs: map[string]*subscriptionInfo{},
	}
}

type subscriptionsChangeFeedResponder struct {
	mu                      sync.RWMutex
	lastChangefeedProcessed atomic.Value // time.Time

	subs map[string]*subscriptionInfo
}

func (c *subscriptionsChangeFeedResponder) GetSubscription(id string) (*subscriptionInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.subs[id]
	return s, ok
}

func (c *subscriptionsChangeFeedResponder) GetCacheSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.subs)
}

func (c *subscriptionsChangeFeedResponder) Lock() {
	c.mu.Lock()
}
func (c *subscriptionsChangeFeedResponder) Unlock() {
	c.mu.Unlock()
}

// the changefeed tracks the OpenShiftClusters change feed and keeps mon.docs
// up-to-date.  We don't monitor clusters in Creating state, hence we don't add
// them to mon.docs.  We also don't monitor clusters in Deleting state; when
// this state is reached we delete from mon.docs
func (r *subscriptionsChangeFeedResponder) OnDoc(sub *api.SubscriptionDocument) {
	id := strings.ToLower(sub.ID)

	// Don't keep subscriptions that are restricted, warned, or are
	// being deleted from our db
	if sub.Subscription.State == api.SubscriptionStateSuspended ||
		sub.Subscription.State == api.SubscriptionStateWarned ||
		sub.Subscription.State == api.SubscriptionStateDeleted {
		// delete is a no-op if it doesn't exist
		delete(r.subs, id)
		return
	}
	c, ok := r.subs[id]
	if ok {
		// update this as subscription might have moved tenants
		c.TenantID = strings.ToLower(sub.Subscription.Properties.TenantID)
	} else {
		r.subs[id] = &subscriptionInfo{
			TenantID: strings.ToLower(sub.Subscription.Properties.TenantID),
		}
	}
}

func (c *subscriptionsChangeFeedResponder) OnAllPendingProcessed() {
	c.lastChangefeedProcessed.Store(time.Now())
}
