package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
)

type openShiftClusterCache struct {
	log *logrus.Entry

	clusters       map[string]*selectorData
	lastChangefeed atomic.Value //time.Time

	mu sync.RWMutex
}

func (c *openShiftClusterCache) Lock() {
	c.mu.Lock()
}
func (c *openShiftClusterCache) Unlock() {
	c.mu.Unlock()
}

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

		delete(c.clusters, id)

	default:
		// Update the selector cache with the cluster data
		data, ok := c.clusters[id]
		if !ok {
			data = &selectorData{}
		}

		err := toSelectorData(doc, data)
		if err != nil {
			c.log.Errorf("failed creating selector data for %s: %s", id, err.Error())
			return
		}
		c.clusters[id] = data
	}
}

func (c *openShiftClusterCache) OnAllPendingProcessed() {
	c.lastChangefeed.Store(time.Now())
}

func toSelectorData(doc *api.OpenShiftClusterDocument, selectorData *selectorData) error {
	resourceID := strings.ToLower(doc.OpenShiftCluster.ID)

	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	selectorData.ResourceID = resourceID
	selectorData.SubscriptionID = r.SubscriptionID

	return nil
}
