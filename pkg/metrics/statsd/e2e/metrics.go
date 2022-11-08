package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

const (
	DimensionARMGeoLocation = "armGeoLocation"
	DimensionARMResourceID  = "armResourceID"
	DimensionResourceType   = "resourceType"
	DimensionSucceeded      = "E2ETestSuccessful"

	LogEntryE2ETestName          = "E2ETestName"
	LogEntryIsE2EEmittableMetric = "IsE2EEmittableMetric"
	LogEntryIsE2ETestSuccessful  = "IsE2ETestSuccessful"
)

type E2ELogToMetrics struct {
	metricsEmitter metrics.Emitter
}

func NewE2ELogToMetrics(m metrics.Emitter) E2ELogToMetrics {
	return E2ELogToMetrics{
		metricsEmitter: m,
	}
}

func (e *E2ELogToMetrics) PostMetricsFromLogEntry(entry *logrus.Entry) {
	if _, ok := entry.Data[LogEntryIsE2EEmittableMetric]; ok {
		metricName := fmt.Sprint(entry.Data[LogEntryE2ETestName])
		metricIsE2ETestSuccessful := fmt.Sprint(entry.Data[LogEntryIsE2ETestSuccessful])
		metricIsE2ETestSuccessfulInt := btoi(strings.EqualFold(metricIsE2ETestSuccessful, "true"))
		dimensions := map[string]string{
			DimensionARMResourceID:  fmt.Sprint(entry.Data[DimensionARMResourceID]),
			DimensionARMGeoLocation: fmt.Sprint(entry.Data[DimensionARMGeoLocation]),
			DimensionResourceType:   fmt.Sprint(entry.Data[DimensionResourceType]),
			DimensionSucceeded:      metricIsE2ETestSuccessful,
		}
		e.metricsEmitter.EmitGauge(metricName, metricIsE2ETestSuccessfulInt, dimensions)
	}
}

func btoi(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
