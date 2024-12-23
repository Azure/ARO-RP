package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmitter(t *testing.T) {
	a := assert.New(t)

	// emitted metrics should line up
	e := NewFakeEmitter()
	e.EmitFloat("foo", 1.1, map[string]string{"bar": "baz"})
	e.EmitGauge("foo", 1, map[string]string{"baz": "bar"})

	errs := e.VerifyEmittedMetrics([]FloatMetric{
		Metric("foo", 1.1, map[string]string{"bar": "baz"}),
	}, []GaugeMetric{
		Metric[int64]("foo", 1, map[string]string{"baz": "bar"}),
	})

	a.Len(errs, 0, "did not match")

	// Unexpected metrics
	errs = e.VerifyEmittedMetrics([]FloatMetric{}, []GaugeMetric{})
	a.Len(errs, 4, "did not match")

	errstrings := make([]string, 0)
	for _, i := range errs {
		errstrings = append(errstrings, i.Error())
	}
	a.Equal(errstrings, []string{
		"expected 0 floats, got 1 instead",
		"expected 0 gauges, got 1 instead",
		"excess float metric: foo = 1.100000 map[bar:baz]",
		"excess gauge metric: foo = 1 map[baz:bar]",
	})

	// Metrics that can't be found
	errs = e.VerifyEmittedMetrics([]FloatMetric{
		Metric("foo", 1.2, map[string]string{"bar": "baz"}),
	}, []GaugeMetric{
		Metric[int64]("foo", 2, map[string]string{"baz": "bar"}),
	})

	errstrings = make([]string, 0)
	for _, i := range errs {
		errstrings = append(errstrings, i.Error())
	}
	a.Equal(errstrings, []string{
		"did not find float foo = 1.200000 map[bar:baz]",
		"did not find gauge foo = 2 map[baz:bar]",
		"excess float metric: foo = 1.100000 map[bar:baz]",
		"excess gauge metric: foo = 1 map[baz:bar]",
	})

	// Reset, we should have none
	e.Reset()
	errs = e.VerifyEmittedMetrics([]FloatMetric{}, []GaugeMetric{})
	a.Len(errs, 0, "did not match")
}
