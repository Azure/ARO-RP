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
	e := NewFakeEmitter(nil)
	e.EmitFloat("foo", 1.1, map[string]string{"bar": "baz"})
	e.EmitGauge("foo", 1, map[string]string{"baz": "bar"})

	errs := e._verifyEmittedMetrics(
		Metric("foo", 1.1, map[string]string{"bar": "baz"}),
		Metric[int64]("foo", 1, map[string]string{"baz": "bar"}),
	)

	a.Len(errs, 0, "did not match")

	// Unexpected metrics
	errs = e._verifyEmittedMetrics()
	a.Len(errs, 3, "did not match")

	errstrings := make([]string, 0)
	for _, i := range errs {
		errstrings = append(errstrings, i.Error())
	}
	a.Equal(errstrings, []string{
		"expected 0 metrics, got 2 instead",
		"excess float metric: foo = 1.100000 map[bar:baz]",
		"excess gauge metric: foo = 1 map[baz:bar]",
	})

	// Metrics that can't be found
	errs = e._verifyEmittedMetrics(
		Metric("foo", 1.2, map[string]string{"bar": "baz"}),
		Metric[int64]("foo", 2, map[string]string{"baz": "bar"}),
	)

	errstrings = make([]string, 0)
	for _, i := range errs {
		errstrings = append(errstrings, i.Error())
	}
	a.Equal(errstrings, []string{
		"did not find metric foo = &matchers.EqualMatcher{Expected:1.2} map[bar:baz]",
		"did not find metric foo = &matchers.EqualMatcher{Expected:2} map[baz:bar]",
		"excess float metric: foo = 1.100000 map[bar:baz]",
		"excess gauge metric: foo = 1 map[baz:bar]",
	})

	// Reset, we should have none
	e.Reset()
	errs = e._verifyEmittedMetrics()
	a.Len(errs, 0, "did not match")
}
