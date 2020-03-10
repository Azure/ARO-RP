package k8s

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/url"
	"time"

	k8smetrics "k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

type metric struct {
	t    time.Time
	name string
}

var _ k8smetrics.LatencyMetric = (*tracer)(nil)
var _ k8smetrics.ResultMetric = (*tracer)(nil)

type tracer struct {
	m metrics.Interface
}

func NewLatency(m metrics.Interface) k8smetrics.LatencyMetric {
	return &tracer{
		m: m,
	}
}

func NewResult(m metrics.Interface) k8smetrics.ResultMetric {
	return &tracer{
		m: m,
	}
}

func (t *tracer) Observe(verb string, url url.URL, latency time.Duration) {
	t.m.EmitGauge("client.k8s.duration", latency.Milliseconds(), map[string]string{
		"path": url.Path,
		"verb": verb,
	})
}

func (t *tracer) Increment(code string, verb string, host string) {
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
