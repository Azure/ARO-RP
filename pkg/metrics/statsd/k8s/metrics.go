package k8s

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/url"
	"time"

	kmetrics "k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

var (
	_ kmetrics.LatencyMetric = (*tracer)(nil)
	_ kmetrics.ResultMetric  = (*tracer)(nil)
)

type tracer struct {
	m metrics.Emitter
}

func NewLatency(m metrics.Emitter) kmetrics.LatencyMetric {
	return &tracer{
		m: m,
	}
}

func NewResult(m metrics.Emitter) kmetrics.ResultMetric {
	return &tracer{
		m: m,
	}
}

func (t *tracer) Observe(ctx context.Context, verb string, url url.URL, latency time.Duration) {
	t.m.EmitGauge("client.k8s.duration", latency.Milliseconds(), map[string]string{
		"path": url.Path,
		"verb": verb,
	})
}

func (t *tracer) Increment(ctx context.Context, code string, verb string, host string) {
	t.m.EmitGauge("client.k8s.count", 1, map[string]string{
		"verb": verb,
		"code": code,
	})

	if code == "<error>" {
		t.m.EmitGauge("client.k8s.errors", 1, map[string]string{
			"verb": verb,
			"code": code,
		})
	}
}
