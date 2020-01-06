package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Interface represents metrics interface
type Interface interface {
	Close()
	EmitFloat(string, float64, ...string) error
	EmitGauge(string, int64, ...string) error
}
