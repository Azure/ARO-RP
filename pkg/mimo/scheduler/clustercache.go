package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"iter"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
)

type openShiftClusterCache struct {
	log *logrus.Entry

	subCache changefeed.SubscriptionsCache

	clusters                   *xsync.Map[string, selectorData]
	lastChangefeed             atomic.Value // time.Time
	initialPopulationWaitGroup *sync.WaitGroup
}

func newOpenShiftClusterCache(log *logrus.Entry, subCache changefeed.SubscriptionsCache) *openShiftClusterCache {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &openShiftClusterCache{
		log:                        log,
		subCache:                   subCache,
		clusters:                   xsync.NewMap[string, selectorData](),
		initialPopulationWaitGroup: wg,
	}
}

// before we accept any, ensure that the subcache is populated by waiting on its
// WaitGroup
func (c *openShiftClusterCache) Lock() {
	c.subCache.WaitForInitialPopulation()
}
func (c *openShiftClusterCache) Unlock() {}

func (c *openShiftClusterCache) GetLastProcessed() (time.Time, bool) {
	t, ok := c.lastChangefeed.Load().(time.Time)
	return t, ok
}

func (c *openShiftClusterCache) OnDoc(doc *api.OpenShiftClusterDocument) {
	id := strings.ToLower(doc.OpenShiftCluster.ID)
	ps := doc.OpenShiftCluster.Properties.ProvisioningState
	fps := doc.OpenShiftCluster.Properties.FailedProvisioningState

	switch {
	case ps == api.ProvisioningStateCreating,
		ps == api.ProvisioningStateDeleting,
		ps == api.ProvisioningStateFailed &&
			(fps == api.ProvisioningStateCreating ||
				fps == api.ProvisioningStateDeleting):

		c.clusters.Delete(id)

	default:
		// Update the selector cache with the cluster data
		c.clusters.Compute(
			id, func(oldValue selectorData, loaded bool) (selectorData, xsync.ComputeOp) {
				new, updated, err := c.toSelectorData(doc, oldValue)
				if err != nil {
					c.log.Errorf("failed creating selector data for %s: %s", id, err.Error())
					return selectorData{}, xsync.CancelOp
				}

				if updated {
					return new, xsync.UpdateOp
				} else {
					return selectorData{}, xsync.CancelOp
				}
			})
	}
}

func (c *openShiftClusterCache) OnAllPendingProcessed() {
	old := c.lastChangefeed.Swap(time.Now())
	// we've done one rotation, unlock the waitgroup
	if old == nil {
		c.initialPopulationWaitGroup.Done()
	}
}

func (c *openShiftClusterCache) toSelectorData(doc *api.OpenShiftClusterDocument, old selectorData) (selectorData, bool, error) {
	new := selectorData{}

	resourceID := strings.ToLower(doc.OpenShiftCluster.ID)

	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, false, err
	}

	new[SelectorDataKeyResourceID] = resourceID
	new[SelectorDataKeySubscriptionID] = r.SubscriptionID

	subCacheData, hasSubCacheData := c.subCache.GetSubscription(r.SubscriptionID)
	if hasSubCacheData {
		new[SelectorDataKeySubscriptionState] = string(subCacheData.State)
	} else {
		return nil, false, fmt.Errorf("no matching subscription %s", r.SubscriptionID)
	}

	return new, !reflect.DeepEqual(old, new), nil
}

func (c *openShiftClusterCache) GetClusters() iter.Seq2[string, selectorData] {
	c.initialPopulationWaitGroup.Wait()
	return c.clusters.All()
}
