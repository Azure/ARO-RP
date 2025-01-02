package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_clusterdata "github.com/Azure/ARO-RP/pkg/util/mocks/clusterdata"
	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestEnrichOne(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	enricherName := "enricherName"

	for _, tt := range []struct {
		name                 string
		failedEnrichers      map[string]bool
		taskCount            int
		taskDuration         int
		timeoutCount         int
		errorCount           int
		enricherCallCount    int
		enricherReturnValue  error
		enricherIsNil        bool
		usesWorkloadIdentity bool
	}{
		{
			name:                "all enrichers called for service principal cluster",
			enricherCallCount:   2,
			enricherReturnValue: nil,
			taskCount:           2,
			taskDuration:        2,
			failedEnrichers:     map[string]bool{enricherName: false},
		},
		{
			name:                 "service principal enricher skipped for workload identity cluster",
			enricherCallCount:    1,
			enricherReturnValue:  nil,
			taskCount:            1,
			taskDuration:         1,
			failedEnrichers:      map[string]bool{enricherName: false},
			usesWorkloadIdentity: true,
		},
		{
			name:                "enricher not called because failed",
			enricherCallCount:   1,
			enricherReturnValue: nil,
			taskCount:           1,
			taskDuration:        1,
			failedEnrichers:     map[string]bool{enricherName: true},
		},
		{
			//should just not panic
			name:                "enricher not called because nil",
			enricherCallCount:   1,
			enricherReturnValue: nil,
			taskCount:           1,
			taskDuration:        1,
			failedEnrichers:     map[string]bool{enricherName: false},
			enricherIsNil:       true,
		},
		{
			name:                "enricher timeout",
			enricherCallCount:   2,
			enricherReturnValue: context.DeadlineExceeded,
			failedEnrichers:     map[string]bool{enricherName: false},
			taskCount:           2,
			taskDuration:        2,
			timeoutCount:        1,
		},
		{
			name:                "enricher error",
			enricherCallCount:   2,
			enricherReturnValue: errors.New("some error"),
			failedEnrichers:     map[string]bool{enricherName: false},
			taskCount:           2,
			taskDuration:        2,
			errorCount:          2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := testmonitor.NewFakeEmitter(t)

			enricherMock := mock_clusterdata.NewMockClusterEnricher(controller)
			enricherMock.EXPECT().Enrich(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tt.enricherReturnValue).Times(tt.enricherCallCount)
			enricherMock.EXPECT().SetDefaults(gomock.Any()).Times(tt.enricherCallCount)

			e := ParallelEnricher{
				emitter: m,
				enrichers: map[string]ClusterEnricher{
					enricherName:     enricherMock,
					servicePrincipal: enricherMock,
				},
				metricsWG: &sync.WaitGroup{},
			}
			if tt.enricherIsNil {
				e.enrichers[enricherName] = nil
			}

			oc := &api.OpenShiftCluster{}
			if tt.usesWorkloadIdentity {
				oc.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{}
			} else {
				oc.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{}
			}

			ctx := context.Background()
			e.enrichOne(ctx, log, oc, clients{}, tt.failedEnrichers)

			e.metricsWG.Wait()

			emittedMetrics := make([]testmonitor.ExpectedMetric, 0)
			for i := 0; i < tt.taskCount; i++ {
				emittedMetrics = append(emittedMetrics, testmonitor.Metric("enricher.tasks.count", int64(1), nil))
			}
			for i := 0; i < tt.taskDuration; i++ {
				emittedMetrics = append(emittedMetrics, testmonitor.MatchingMetric("enricher.tasks.duration", gomega.BeNumerically(">", -1), map[string]string{"task": "*mock_clusterdata.MockClusterEnricher"}))
			}
			for i := 0; i < tt.timeoutCount; i++ {
				emittedMetrics = append(emittedMetrics, testmonitor.Metric("enricher.timeouts", int64(1), nil))
			}
			for i := 0; i < tt.errorCount; i++ {
				emittedMetrics = append(emittedMetrics, testmonitor.Metric("enricher.tasks.errors", int64(1), nil))
			}

			m.VerifyEmittedMetrics(emittedMetrics...)
		})
	}
}
