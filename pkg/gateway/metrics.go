package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"sync/atomic"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/recover"
)

func (g *gateway) emitMetrics() {
	defer recover.Panic(g.log)

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for range t.C {
		g._emitMetrics()
	}
}

func (g *gateway) _emitMetrics() {
	g.m.EmitGauge("gateway.connections.open", atomic.LoadInt64(&g.httpConnections), map[string]string{
		"protocol": "http",
	})

	g.m.EmitGauge("gateway.connections.open", atomic.LoadInt64(&g.httpsConnections), map[string]string{
		"protocol": "https",
	})

	if lastChangefeed, ok := g.lastChangefeed.Load().(time.Time); ok {
		g.m.EmitGauge("gateway.lastchangefeed", lastChangefeed.Unix(), nil)
	}
}
