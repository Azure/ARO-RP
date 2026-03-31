package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailsIfNotAssertedOnCleanup(t *testing.T) {
	b := &bytes.Buffer{}
	it := &testing.T{}
	m := NewFakeMetricsEmitter(it)
	m.testOutput = b
	require.False(t, it.Failed())
	m.onCleanup()
	require.True(t, it.Failed(), "should have caused the test to fail")
	require.Equal(t,
		"!!! did not assert on any metric floats !!!\n!!! did not assert on any metric gauges !!!\n",
		b.String())
}

func TestAssertedOnCleanupDoesNotFail(t *testing.T) {
	b := &bytes.Buffer{}
	it := &testing.T{}
	m := NewFakeMetricsEmitter(it)
	m.testOutput = b
	require.False(t, it.Failed())
	m.AssertFloats()
	m.AssertGauges()
	m.onCleanup()
	require.False(t, it.Failed(), "should not have caused the test to fail")
	require.Empty(t, b.String())
}

func TestAssertChecks(t *testing.T) {
	testCases := []struct {
		desc           string
		emit           func(m *fakeMetricsEmitter)
		expectedGauges []MetricsAssertion[int64]
		expectedFloats []MetricsAssertion[float64]
		testShouldFail bool
		testOutput     []string
	}{
		{
			desc: "checks for gauge metrics with or without dims",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitGauge("testmetric", 22, nil)
				m.EmitGauge("testmetric", 22, map[string]string{"system": "on"})
			},
			expectedGauges: []MetricsAssertion[int64]{},
			expectedFloats: []MetricsAssertion[float64]{},
			testShouldFail: true,
			testOutput: []string{
				"gauge metric 'testmetric' with dims 'map[]' not asserted upon",
				"gauge metric 'testmetric' with dims 'map[system:on]' not asserted upon",
			},
		},
		{
			desc: "checks for float metrics with or without dims",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitFloat("testmetric", 22.0, nil)
				m.EmitFloat("testmetric", 22.0, map[string]string{"system": "on"})
			},
			expectedGauges: []MetricsAssertion[int64]{},
			expectedFloats: []MetricsAssertion[float64]{},
			testShouldFail: true,
			testOutput: []string{
				"float metric 'testmetric' with dims 'map[]' not asserted upon",
				"float metric 'testmetric' with dims 'map[system:on]' not asserted upon",
			},
		},
		{
			desc: "matching only one gauge still errors",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitGauge("testmetric", 22, nil)
				m.EmitGauge("testmetric", 22, map[string]string{"system": "on"})
			},
			expectedGauges: []MetricsAssertion[int64]{
				{MetricName: "testmetric", Dimensions: map[string]string{}, Value: 22},
			},
			expectedFloats: []MetricsAssertion[float64]{},
			testShouldFail: true,
			testOutput: []string{
				"gauge metric 'testmetric' with dims 'map[system:on]' not asserted upon",
			},
		},
		{
			desc: "looking for a non-existant gauge fails",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitGauge("testmetric", 22, nil)
			},
			expectedGauges: []MetricsAssertion[int64]{
				{MetricName: "testmetric", Dimensions: map[string]string{}, Value: 22},
				{MetricName: "othermetric", Dimensions: map[string]string{}, Value: 22},
			},
			expectedFloats: []MetricsAssertion[float64]{},
			testShouldFail: true,
			testOutput: []string{
				"gauge metric 'othermetric' with dims 'map[]' was not emitted",
			},
		},
		{
			desc: "looking for a gauge with the wrong value fails",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitGauge("testmetric", 22, nil)
			},
			expectedGauges: []MetricsAssertion[int64]{
				{MetricName: "testmetric", Dimensions: nil, Value: 23},
			},
			expectedFloats: []MetricsAssertion[float64]{},
			testShouldFail: true,
			testOutput: []string{
				"gauge metric 'testmetric' with dims 'map[]' had incorrect emitted value 22, wanted 23",
			},
		},
		{
			desc: "looking for a float with the wrong value fails",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitFloat("testmetric", 22.0, nil)
			},
			expectedGauges: []MetricsAssertion[int64]{},
			expectedFloats: []MetricsAssertion[float64]{
				{MetricName: "testmetric", Dimensions: nil, Value: 23.0},
			},
			testShouldFail: true,
			testOutput: []string{
				"float metric 'testmetric' with dims 'map[]' had incorrect emitted value 22.000000, wanted 23.000000",
			},
		},
		{
			desc: "looking for a non-existant float fails",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitFloat("testmetric", 22.11, nil)
			},
			expectedGauges: []MetricsAssertion[int64]{},
			expectedFloats: []MetricsAssertion[float64]{
				{MetricName: "testmetric", Dimensions: map[string]string{}, Value: 22.11},
				{MetricName: "othermetric", Dimensions: map[string]string{}, Value: 22.11},
			},
			testShouldFail: true,
			testOutput: []string{
				"float metric 'othermetric' with dims 'map[]' was not emitted",
			},
		},
		{
			desc: "matching both gauges passes",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitGauge("testmetric", 22, nil)
				m.EmitGauge("testmetric", 22, map[string]string{"system": "on"})
			},
			expectedGauges: []MetricsAssertion[int64]{
				{MetricName: "testmetric", Dimensions: map[string]string{}, Value: 22},
				{MetricName: "testmetric", Dimensions: map[string]string{"system": "on"}, Value: 22},
			},
			expectedFloats: []MetricsAssertion[float64]{},
			testShouldFail: false,
			testOutput:     []string{},
		},
		{
			desc: "matching both floats passes",
			emit: func(m *fakeMetricsEmitter) {
				m.EmitFloat("testmetric", 22.11, nil)
				m.EmitFloat("testmetric", 22.11, map[string]string{"system": "on"})
			},
			expectedGauges: []MetricsAssertion[int64]{},
			expectedFloats: []MetricsAssertion[float64]{
				{MetricName: "testmetric", Dimensions: map[string]string{}, Value: 22.11},
				{MetricName: "testmetric", Dimensions: map[string]string{"system": "on"}, Value: 22.11},
			},
			testShouldFail: false,
			testOutput:     []string{},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			b := &bytes.Buffer{}
			it := &testing.T{}
			m := NewFakeMetricsEmitter(it)
			m.testOutput = b
			require.False(t, it.Failed())

			// add some metrics, if we have any
			tt.emit(m)

			m.AssertFloats(tt.expectedFloats...)
			m.AssertGauges(tt.expectedGauges...)
			m.onCleanup()
			require.Equal(t, tt.testShouldFail, it.Failed())
			var output []string
			if b.String() != "" {
				output = strings.Split(strings.TrimSpace(b.String()), "\n")
			}
			require.ElementsMatch(t, tt.testOutput, output)
		})
	}
}
