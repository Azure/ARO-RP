package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"

	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestEmitMetrics(t *testing.T) {
	testStartTime := time.Now()

	for _, tt := range []struct {
		name               string
		httpConnections    int64
		httpsConnections   int64
		lastChangefeedTime time.Time
	}{
		{
			name:               "1 http connection 1 https connection no lastChangefeed",
			httpConnections:    1,
			httpsConnections:   1,
			lastChangefeedTime: testStartTime,
		},
		{
			name:               "0 http connection 0 https connections lastChangefeed loads",
			lastChangefeedTime: time.Date(2022, time.April, 19, 9, 0, 0, 0, time.UTC),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			emitter := testmonitor.NewFakeEmitter(t)

			expectedGauges := []testmonitor.ExpectedMetric{
				testmonitor.Metric("gateway.connections.open", tt.httpConnections, map[string]string{"protocol": "http"}),
				testmonitor.Metric("gateway.connections.open", tt.httpsConnections, map[string]string{"protocol": "https"}),
			}

			gateway := gateway{
				m:                emitter,
				httpConnections:  tt.httpConnections,
				httpsConnections: tt.httpsConnections,
			}

			if !tt.lastChangefeedTime.Equal(testStartTime) {
				gateway.lastChangefeed.Store(tt.lastChangefeedTime)
				expectedGauges = append(expectedGauges, testmonitor.Metric("gateway.lastchangefeed", tt.lastChangefeedTime.Unix(), nil))
			}

			gateway._emitMetrics()

			emitter.VerifyEmittedMetrics(expectedGauges...)
		})
	}
}
