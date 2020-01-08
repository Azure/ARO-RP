package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"testing"
	"time"
)

const (
	testKey     = "tests.test_key"
	testOutputF = "{\"Metric\":\"tests.test_key\",\"Account\":\"test_account\",\"Namespace\":\"test_namespace\",\"Dims\":{\"key\":\"value\"},\"TS\":\"0001-01-01T00:00:00.000\"}:%f|f\n"
	testOutputG = "{\"Metric\":\"tests.test_key\",\"Account\":\"test_account\",\"Namespace\":\"test_namespace\",\"Dims\":{\"key\":\"value\"},\"TS\":\"0001-01-01T00:00:00.000\"}:%d|g\n"
)

type writeCloser struct {
	*bytes.Buffer
}

func (c *writeCloser) Close() error { return nil }
func TestEmitGauge(t *testing.T) {
	wc := &writeCloser{Buffer: &bytes.Buffer{}}
	c := &Statsd{
		conn: wc,
		now:  func() time.Time { return time.Time{} },
	}

	err := c.EmitGauge(testKey, 42, map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if wc.String() != "{\"Metric\":\"tests.test_key\",\"Dims\":{\"key\":\"value\"},\"TS\":\"0001-01-01T00:00:00.000\"}:42|g\n" {
		t.Error(wc.String())
	}

}

func TestEmitFloat(t *testing.T) {
	wc := &writeCloser{Buffer: &bytes.Buffer{}}
	c := &Statsd{
		conn: wc,
		now:  func() time.Time { return time.Time{} },
	}
	err := c.EmitFloat(testKey, 5, map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if wc.String() != "{\"Metric\":\"tests.test_key\",\"Dims\":{\"key\":\"value\"},\"TS\":\"0001-01-01T00:00:00.000\"}:5.000000|f\n" {
		t.Error(wc.String())
	}
}
