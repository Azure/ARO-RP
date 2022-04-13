package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Emitter emits different types of metrics
type Emitter interface {
	EmitFloat(string, float64, map[string]string)
	EmitGauge(string, int64, map[string]string)
}
