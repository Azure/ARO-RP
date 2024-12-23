package azure

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/onsi/gomega"

	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestTracer(t *testing.T) {
	m := testmonitor.NewFakeEmitter(t)

	tr := New(m)
	ctx := tr.StartSpan(context.Background(), "test")
	tr.EndSpan(ctx, http.StatusUnauthorized, errors.New("authorization failed"))

	m.VerifyEmittedMetrics(
		testmonitor.MatchingMetric("client.azure.duration", gomega.BeNumerically(">", -0.01), map[string]string{
			"client": "test",
			"code":   "401",
		}),
		testmonitor.Metric("client.azure.count", int64(1), map[string]string{
			"client": "test",
			"code":   "401",
		}),
		testmonitor.Metric("client.azure.errors", int64(1), map[string]string{
			"client": "test",
			"code":   "401",
		}),
	)
}
