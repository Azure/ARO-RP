package noop

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Noop struct{}

func (c *Noop) Close() error {
	return nil
}

func (c *Noop) EmitFloat(stat string, value float64, dims map[string]string) error {
	return nil
}

func (c *Noop) EmitGauge(stat string, value int64, dims map[string]string) error {
	return nil
}
