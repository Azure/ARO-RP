package k8s

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestTracer(t *testing.T) {
	m := testmonitor.NewFakeEmitter(t)

	tr := &tracer{
		m: m,
	}
	tr.Increment(context.Background(), "<error>", "GET", "host")

	m.VerifyEmittedMetrics(
		testmonitor.Metric("client.k8s.count", int64(1), map[string]string{
			"verb": "GET",
			"code": "<error>",
		}),
		testmonitor.Metric("client.k8s.errors", int64(1), map[string]string{
			"verb": "GET",
			"code": "<error>",
		}),
	)
}
