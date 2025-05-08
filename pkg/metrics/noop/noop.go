package noop

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Noop struct{}

func (c *Noop) EmitFloat(metricName string, metricValue float64, dimensions map[string]string) {}
func (c *Noop) EmitGauge(metricName string, metricValue int64, dimensions map[string]string)   {}
