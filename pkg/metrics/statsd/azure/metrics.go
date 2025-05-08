package azure

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Azure/go-autorest/tracing"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

var _ tracing.Tracer = (*tracer)(nil)

type tracer struct {
	m metrics.Emitter
}

type contextKey int

const (
	contextKeyMetric contextKey = iota
)

type metric struct {
	t    time.Time
	name string
}

func New(m metrics.Emitter) tracing.Tracer {
	return &tracer{
		m: m,
	}
}

func (t *tracer) NewTransport(base *http.Transport) http.RoundTripper {
	return base
}

func (t *tracer) StartSpan(ctx context.Context, name string) context.Context {
	start := time.Now()
	return context.WithValue(ctx, contextKeyMetric, metric{
		name: name,
		t:    start,
	})
}

func (t *tracer) EndSpan(ctx context.Context, httpStatusCode int, err error) {
	metric := ctx.Value(contextKeyMetric).(metric)

	t.m.EmitGauge("client.azure.duration", time.Since(metric.t).Milliseconds(), map[string]string{
		"client": metric.name,
		"code":   strconv.Itoa(httpStatusCode),
	})

	t.m.EmitGauge("client.azure.count", 1, map[string]string{
		"client": metric.name,
		"code":   strconv.Itoa(httpStatusCode),
	})

	if err != nil {
		t.m.EmitGauge("client.azure.errors", 1, map[string]string{
			"client": metric.name,
			"code":   strconv.Itoa(httpStatusCode),
		})
	}
}
