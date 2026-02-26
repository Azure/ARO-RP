package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/puzpuzpuz/xsync/v4"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

type fakeMetricsEmitter struct {
	t testing.TB

	gauges           *xsync.Map[string, int64]
	assertedOnGauges bool
	floats           *xsync.Map[string, float64]
	assertedOnFloats bool
}

type MetricsAssertion[X int64 | float64] struct {
	MetricName string
	Dimensions map[string]string
	Value      X
}

type FakeMetrics interface {
	metrics.Emitter

	AssertFloats(...MetricsAssertion[float64])
	AssertGauges(...MetricsAssertion[int64])
	Reset(testing.TB)
}

func NewFakeMetricsEmitter(t testing.TB) *fakeMetricsEmitter {
	m := xsync.NewMap[string, int64]()
	f := xsync.NewMap[string, float64]()

	e := &fakeMetricsEmitter{
		t: t,

		gauges: m,
		floats: f,
	}

	// handler to check we asserted on values
	t.Cleanup(e.onCleanup)

	return e
}

func getKey(metricName string, dims map[string]string) string {
	c := map[string]string{}

	if dims != nil {
		maps.Copy(c, dims)
	}
	c["__METRIC_NAME"] = metricName

	r, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return string(r)
}

func (e *fakeMetricsEmitter) Reset(t testing.TB) {
	e.t = t
	e.assertedOnFloats = false
	e.assertedOnGauges = false
	e.floats.Clear()
	e.gauges.Clear()
}

func (e *fakeMetricsEmitter) onCleanup() {
	if !e.t.Failed() {
		if !e.assertedOnFloats {
			e.t.Error("!!! did not assert on any metric floats !!!")
		}
		if !e.assertedOnGauges {
			e.t.Error("!!! did not assert on any metric gauges !!!")
		}
	}
}

func (e *fakeMetricsEmitter) EmitGauge(metricName string, metricValue int64, dimensions map[string]string) {
	key := getKey(metricName, dimensions)
	e.gauges.Store(key, metricValue)
}

func (e *fakeMetricsEmitter) EmitFloat(metricName string, metricValue float64, dimensions map[string]string) {
	key := getKey(metricName, dimensions)
	e.floats.Store(key, metricValue)
}

func (e *fakeMetricsEmitter) AssertFloats(assertions ...MetricsAssertion[float64]) {
	// check the assertions we have been given
	for _, a := range assertions {
		seekingKey := getKey(a.MetricName, a.Dimensions)

		val, ok := e.floats.LoadAndDelete(seekingKey)
		if !ok {
			e.t.Errorf("float metric '%s' with dims '%s' was not emitted", a.MetricName, a.Dimensions)
		} else {
			if val != a.Value {
				e.t.Errorf("float metric '%s' with dims '%s' had incorrect emitted value %f, wanted %f", a.MetricName, a.Dimensions, val, a.Value)
			}
		}
	}

	for k := range e.floats.All() {
		dims := map[string]string{}
		err := json.Unmarshal([]byte(k), &dims)
		if err != nil {
			e.t.Errorf("failed unmarshalling: %s", err)
		}
		key := dims["__METRIC_NAME"]
		delete(dims, "__METRIC_NAME")
		e.t.Errorf("float metric '%s' with dims '%s' not asserted upon", key, dims)
	}

	e.assertedOnFloats = true
}

func (e *fakeMetricsEmitter) AssertGauges(assertions ...MetricsAssertion[int64]) {
	// check the assertions we have been given
	for _, a := range assertions {
		seekingKey := getKey(a.MetricName, a.Dimensions)

		val, ok := e.gauges.LoadAndDelete(seekingKey)
		if !ok {
			e.t.Errorf("gauge metric '%s' with dims '%s' was not emitted", a.MetricName, a.Dimensions)
		} else {
			if val != a.Value {
				e.t.Errorf("gauge metric '%s' with dims '%s' had incorrect emitted value %d, wanted %d", a.MetricName, a.Dimensions, val, a.Value)
			}
		}
	}

	for k := range e.gauges.All() {
		dims := map[string]string{}
		err := json.Unmarshal([]byte(k), &dims)
		if err != nil {
			e.t.Errorf("failed unmarshalling: %s", err)
		}
		key := dims["__METRIC_NAME"]
		delete(dims, "__METRIC_NAME")
		e.t.Errorf("gauge metric '%s' with dims '%s' not asserted upon", key, dims)
	}

	e.assertedOnGauges = true
}
