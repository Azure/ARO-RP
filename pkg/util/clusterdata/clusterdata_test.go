package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_clusterdata "github.com/Azure/ARO-RP/pkg/util/mocks/clusterdata"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEnrichOne(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	enricherName := "enricherName"

	for _, tt := range []struct {
		name                string
		failedEnrichers     map[string]bool
		taskCount           int
		taskDuration        int
		timeoutCount        int
		errorCount          int
		enricherCallCount   int
		enricherReturnValue error
		enricherIsNil       bool
	}{
		{
			name:                "enricher called",
			enricherCallCount:   1,
			enricherReturnValue: nil,
			taskCount:           1,
			taskDuration:        1,
			failedEnrichers:     map[string]bool{enricherName: false},
		},
		{
			name:              "enricher not called because failed",
			enricherCallCount: 0,
			failedEnrichers:   map[string]bool{enricherName: true},
		},
		{
			//should just not panic
			name:            "enricher not called because nil",
			failedEnrichers: map[string]bool{enricherName: false},
			enricherIsNil:   true,
		},
		{
			name:                "enricher timeout",
			enricherCallCount:   1,
			enricherReturnValue: context.DeadlineExceeded,
			failedEnrichers:     map[string]bool{enricherName: false},
			taskCount:           1,
			taskDuration:        1,
			timeoutCount:        1,
		},
		{
			name:                "enricher error",
			enricherCallCount:   1,
			enricherReturnValue: errors.New("some error"),
			failedEnrichers:     map[string]bool{enricherName: false},
			taskCount:           1,
			taskDuration:        1,
			errorCount:          1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			metricsMock := mock_metrics.NewMockEmitter(controller)
			metricsMock.EXPECT().EmitGauge("enricher.tasks.count", int64(1), nil).Times(tt.taskCount)
			metricsMock.EXPECT().EmitGauge("enricher.tasks.duration", gomock.Any(), gomock.Any()).Times(tt.taskDuration)
			metricsMock.EXPECT().EmitGauge("enricher.timeouts", int64(1), nil).Times(tt.timeoutCount)
			metricsMock.EXPECT().EmitGauge("enricher.tasks.errors", int64(1), nil).Times(tt.errorCount)

			enricherMock := mock_clusterdata.NewMockClusterEnricher(controller)
			enricherMock.EXPECT().Enrich(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tt.enricherReturnValue).Times(tt.enricherCallCount)

			e := ParallelEnricher{
				emitter: metricsMock,
				enrichers: map[string]ClusterEnricher{
					enricherName: enricherMock,
				},
			}
			if tt.enricherIsNil {
				e.enrichers[enricherName] = nil
			}

			ctx := context.Background()
			e.enrichOne(ctx, log, &api.OpenShiftCluster{}, clients{}, tt.failedEnrichers)
		})
	}
}
