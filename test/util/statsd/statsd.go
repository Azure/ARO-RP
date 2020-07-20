package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"time"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

type Interface interface {
	metrics.Interface
	Dump() []string
}

type metric struct {
	metric    string
	account   string
	namespace string
	dims      map[string]string
	ts        time.Time

	valueGauge *int64
	valueFloat *float64
}

type fakeStatsd struct {
	metrics []*metric
	now     func() time.Time
}

func New() *fakeStatsd {
	return &fakeStatsd{
		now: time.Now,
	}
}

func (s *fakeStatsd) emitMetric(m *metric) {
	m.account = "*"
	m.namespace = "*"
	if m.dims == nil {
		m.dims = map[string]string{}
	}
	m.dims["location"] = "test"
	m.dims["hostname"] = "testhost"
	m.ts = s.now()

	s.metrics = append(s.metrics, m)
}

func (s *fakeStatsd) Dump() []string {

	r := make([]string, 0)

	for _, m := range s.metrics {
		if m.valueFloat != nil {
			r = append(r, fmt.Sprintf("%s: value %f", m.metric, *m.valueFloat))
		}
		if m.valueGauge != nil {
			r = append(r, fmt.Sprintf("%s: value %d", m.metric, *m.valueGauge))
		}
	}

	return r
}

// EmitFloat records float information
func (s *fakeStatsd) EmitFloat(m string, value float64, dims map[string]string) {
	s.emitMetric(&metric{
		metric:     m,
		dims:       dims,
		valueFloat: &value,
	})
}

// EmitGauge records gauge information
func (s *fakeStatsd) EmitGauge(m string, value int64, dims map[string]string) {
	s.emitMetric(&metric{
		metric:     m,
		dims:       dims,
		valueGauge: &value,
	})
}
