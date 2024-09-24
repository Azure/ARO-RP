package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
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
			mockController := gomock.NewController(t)
			defer mockController.Finish()
			mock_metrics := mock_metrics.NewMockEmitter(mockController)

			mock_metrics.EXPECT().EmitGauge("gateway.connections.open", tt.httpConnections, map[string]string{"protocol": "http"}).Times(1)
			mock_metrics.EXPECT().EmitGauge("gateway.connections.open", tt.httpsConnections, map[string]string{"protocol": "https"}).Times(1)

			gateway := gateway{
				m:                mock_metrics,
				httpConnections:  tt.httpConnections,
				httpsConnections: tt.httpsConnections,
			}

			if !tt.lastChangefeedTime.Equal(testStartTime) {
				gateway.lastChangefeed.Store(tt.lastChangefeedTime)
				mock_metrics.EXPECT().EmitGauge("gateway.lastchangefeed", tt.lastChangefeedTime.Unix(), nil).Times(1)
			}

			gateway._emitMetrics()
		})
	}
}
