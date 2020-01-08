package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Interface represents metrics interface
type Interface interface {
	Close() error
	EmitFloat(string, float64, map[string]string) error
	EmitGauge(string, int64, map[string]string) error
}
