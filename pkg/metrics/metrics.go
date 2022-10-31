package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Emitter emits different types of metrics
type Emitter interface {
	EmitFloat(metricName string, metricValue float64, dimensions map[string]string)
	EmitGauge(metricName string, metricValue int64, dimensions map[string]string)
}
