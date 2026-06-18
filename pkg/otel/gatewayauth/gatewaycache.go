package gatewayauth

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics"
)

type gatewayCache struct {
	log *logrus.Entry
	m   metrics.Emitter

	// map of linkid: cluster resource ID
	clusters *xsync.Map[string, string]

	lastChangefeedDataUpdate   atomic.Value // time.Time
	lastChangefeedProcessed    atomic.Value // time.Time
	initialPopulationWaitGroup *sync.WaitGroup
}

func newGatewayCache(log *logrus.Entry, m metrics.Emitter) *gatewayCache {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &gatewayCache{
		log:                        log,
		m:                          m,
		clusters:                   xsync.NewMap[string, string](),
		initialPopulationWaitGroup: wg,
	}
}

func (c *gatewayCache) Lock() {
}
func (c *gatewayCache) Unlock() {}

func (c *gatewayCache) GetLastProcessed() (time.Time, bool) {
	t, ok := c.lastChangefeedProcessed.Load().(time.Time)
	return t, ok
}

func (c *gatewayCache) GetLastDataUpdate() (time.Time, bool) {
	t, ok := c.lastChangefeedDataUpdate.Load().(time.Time)
	return t, ok
}

func (c *gatewayCache) OnDoc(doc *api.GatewayDocument) {
	c.clusters.Compute(doc.ID, func(oldValue string, loaded bool) (string, xsync.ComputeOp) {
		if doc.Gateway.Deleting {
			return oldValue, xsync.DeleteOp
		} else {
			if loaded {
				// don't do anything if it already exists
				return oldValue, xsync.CancelOp
			} else {
				return strings.ToLower(doc.Gateway.ID), xsync.UpdateOp
			}
		}
	})
}

func (c *gatewayCache) OnAllPendingProcessed(gotAny bool) {
	now := time.Now()
	old := c.lastChangefeedProcessed.Swap(now)
	if gotAny {
		c.lastChangefeedDataUpdate.Store(now)
	}
	// we've done one rotation, unlock the waitgroup
	if old == nil {
		defer c.initialPopulationWaitGroup.Done()
	}
	c.m.EmitGauge("changefeed.caches.size", int64(c.clusters.Size()), map[string]string{
		"name": "GatewayDocument",
	})
}
