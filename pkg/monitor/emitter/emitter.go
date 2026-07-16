package emitter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "maps"

import "github.com/Azure/ARO-RP/pkg/metrics"

func EmitGauge(emitter metrics.Emitter, name string, value int64, existing map[string]string, additional map[string]string) {
	if additional == nil {
		additional = map[string]string{}
	}
	maps.Copy(additional, existing)
	emitter.EmitGauge(name, value, additional)
}

func EmitFloat(emitter metrics.Emitter, name string, value float64, existing map[string]string, additional map[string]string) {
	if additional == nil {
		additional = map[string]string{}
	}
	maps.Copy(additional, existing)
	emitter.EmitFloat(name, value, additional)
}
