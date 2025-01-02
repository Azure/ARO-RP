package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"maps"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	gomegatypes "github.com/onsi/gomega/types"

	"golang.org/x/exp/slices"
)

type emittedMetric[T float64 | int64] struct {
	name       string
	value      T
	dimensions map[string]string
}

type FloatMetric = emittedMetric[float64]
type GaugeMetric = emittedMetric[int64]

type ExpectedMetric struct {
	name         string
	valueMatcher gomegatypes.GomegaMatcher
	dimensions   map[string]string
}

func Metric[T float64 | int64](metricName string, metricValue T, dimensions map[string]string) ExpectedMetric {
	return ExpectedMetric{
		name:         metricName,
		valueMatcher: gomega.Equal(metricValue),
		dimensions:   dimensions,
	}
}

func MatchingMetric(metricName string, matcher types.GomegaMatcher, dimensions map[string]string) ExpectedMetric {
	return ExpectedMetric{
		name:         metricName,
		valueMatcher: matcher,
		dimensions:   dimensions,
	}
}

type fakeEmitter struct {
	t        *testing.T
	asserted bool

	floats []emittedMetric[float64]
	gauges []emittedMetric[int64]
}

func NewFakeEmitter(t *testing.T) *fakeEmitter {
	e := &fakeEmitter{
		t:      t,
		floats: make([]emittedMetric[float64], 0),
		gauges: make([]emittedMetric[int64], 0),
	}

	if t != nil {
		t.Cleanup(func() {
			if !e.asserted {
				t.Error("metrics were not asserted upon")
			}
		})
	}

	return e
}

func (c *fakeEmitter) EmitFloat(metricName string, metricValue float64, dimensions map[string]string) {
	c.floats = append(c.floats, emittedMetric[float64]{
		name:       metricName,
		value:      metricValue,
		dimensions: dimensions,
	})
}

func (c *fakeEmitter) EmitGauge(metricName string, metricValue int64, dimensions map[string]string) {
	c.gauges = append(c.gauges, emittedMetric[int64]{
		name:       metricName,
		value:      metricValue,
		dimensions: dimensions,
	})
}

func (c *fakeEmitter) Reset() {
	c.floats = make([]emittedMetric[float64], 0)
	c.gauges = make([]emittedMetric[int64], 0)
	c.asserted = false
}

// VerifyEmittedMetrics will verify the output of the emitter with a list of
// expected float and gauge metrics. Metrics can be out of order, but can only
// be consumed once. If you want to check the ordering of metrics (e.g. that a
// number goes up and then down), organise your code such that you can test the
// initial value, use this struct's Reset() method, and then test for the higher
// value.
func (c *fakeEmitter) VerifyEmittedMetrics(metrics ...ExpectedMetric) {
	for _, err := range c._verifyEmittedMetrics(metrics...) {
		c.t.Error(err)
	}
}
func (c *fakeEmitter) _verifyEmittedMetrics(metrics ...ExpectedMetric) []error {
	c.asserted = true
	errors := make([]error, 0)

	if len(metrics) != len(c.floats)+len(c.gauges) {
		errors = append(errors, fmt.Errorf("expected %d metrics, got %d instead", len(metrics), len(c.floats)+len(c.gauges)))
	}

	foundFloats := make([]int, 0)
	foundGauges := make([]int, 0)

	for _, wanted := range metrics {
		found := false
		for x, emitted := range c.floats {

			s, err := wanted.valueMatcher.Match(emitted.value)
			if err != nil {
				errors = append(errors, err)
				return errors
			}

			if slices.Index(foundFloats, x) == -1 &&
				wanted.name == emitted.name &&
				s == true &&
				maps.Equal(wanted.dimensions, emitted.dimensions) {
				found = true
				foundFloats = append(foundFloats, x)
				break
			}
		}

		for x, emitted := range c.gauges {
			s, err := wanted.valueMatcher.Match(emitted.value)
			if err != nil {
				errors = append(errors, err)
				return errors
			}

			if slices.Index(foundGauges, x) == -1 &&
				wanted.name == emitted.name &&
				s == true &&
				maps.Equal(wanted.dimensions, emitted.dimensions) {
				found = true
				foundGauges = append(foundGauges, x)
				break
			}
		}

		if !found {
			errors = append(errors, fmt.Errorf("did not find metric %s = %#+v %s", wanted.name, wanted.valueMatcher, wanted.dimensions))
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
