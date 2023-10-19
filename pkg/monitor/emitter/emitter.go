package emitter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/metrics"

func EmitGauge(emitter metrics.Emitter, name string, value int64, existing map[string]string, additional map[string]string) {
	if additional == nil {
		additional = map[string]string{}
	}
	for k, v := range existing {
		additional[k] = v
	}
	emitter.EmitGauge(name, value, additional)
}
