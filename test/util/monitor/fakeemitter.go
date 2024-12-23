package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"maps"

	"golang.org/x/exp/slices"
)

type emittedMetric[T float64 | int64] struct {
	name       string
	value      T
	dimensions map[string]string
}

type FloatMetric = emittedMetric[float64]
type GaugeMetric = emittedMetric[int64]

func Metric[T float64 | int64](metricName string, metricValue T, dimensions map[string]string) emittedMetric[T] {
	return emittedMetric[T]{
		name:       metricName,
		value:      metricValue,
		dimensions: dimensions,
	}
}

type fakeEmitter struct {
	floats []emittedMetric[float64]
	gauges []emittedMetric[int64]
}

func NewFakeEmitter() *fakeEmitter {
	return &fakeEmitter{
		floats: make([]emittedMetric[float64], 0),
		gauges: make([]emittedMetric[int64], 0),
	}
}

func (c *fakeEmitter) EmitFloat(metricName string, metricValue float64, dimensions map[string]string) {
	c.floats = append(c.floats, Metric(metricName, metricValue, dimensions))
}

func (c *fakeEmitter) EmitGauge(metricName string, metricValue int64, dimensions map[string]string) {
	c.gauges = append(c.gauges, Metric(metricName, metricValue, dimensions))
}

func (c *fakeEmitter) Reset() {
	c.floats = make([]emittedMetric[float64], 0)
	c.gauges = make([]emittedMetric[int64], 0)
}

// VerifyEmittedMetrics will verify the output of the emitter with a list of
// expected float and gauge metrics. Metrics can be out of order, but can only
// be consumed once. If you want to check the ordering of metrics (e.g. that a
// number goes up and then down), organise your code such that you can test the
// initial value, use this struct's Reset() method, and then test for the higher
// value.
func (c *fakeEmitter) VerifyEmittedMetrics(floats []FloatMetric, gauges []GaugeMetric) []error {
	errors := make([]error, 0)

	if len(floats) != len(c.floats) {
		errors = append(errors, fmt.Errorf("expected %d floats, got %d instead", len(floats), len(c.floats)))
	}

	if len(gauges) != len(c.gauges) {
		errors = append(errors, fmt.Errorf("expected %d gauges, got %d instead", len(gauges), len(c.gauges)))
	}
	foundFloats := make([]int, 0)
	foundGauges := make([]int, 0)

	for _, wanted := range floats {
		found := false
		for x, emitted := range c.floats {

			if slices.Index(foundFloats, x) == -1 &&
				wanted.name == emitted.name &&
				wanted.value == emitted.value &&
				maps.Equal(wanted.dimensions, emitted.dimensions) {
				found = true
				foundFloats = append(foundFloats, x)
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Errorf("did not find float %s = %f %s", wanted.name, wanted.value, wanted.dimensions))
		}
	}

	for _, wanted := range gauges {
		found := false
		for x, emitted := range c.gauges {
			if wanted.name == emitted.name &&
				wanted.value == emitted.value &&
				maps.Equal(wanted.dimensions, emitted.dimensions) {
				found = true
				foundGauges = append(foundGauges, x)
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Errorf("did not find gauge %s = %d %s", wanted.name, wanted.value, wanted.dimensions))
		}
	}

	for x, i := range c.floats {
		if slices.Index(foundFloats, x) == -1 {
			errors = append(errors, fmt.Errorf("excess float metric: %s = %f %s", i.name, i.value, i.dimensions))
		}
	}

	for x, i := range c.gauges {
		if slices.Index(foundGauges, x) == -1 {
			errors = append(errors, fmt.Errorf("excess gauge metric: %s = %d %s", i.name, i.value, i.dimensions))
		}
	}

	return errors
}
