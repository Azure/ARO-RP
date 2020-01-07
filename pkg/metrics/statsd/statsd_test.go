package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

const (
	testKey     = "tests.test_key"
	testOutputF = "{\"Metric\":\"tests.test_key\",\"Account\":\"test_account\",\"Namespace\":\"test_namespace\",\"Dims\":{\"key\":\"value\"},\"TS\":\"0001-01-01T00:00:00.000\"}:%f|f\n"
	testOutputG = "{\"Metric\":\"tests.test_key\",\"Account\":\"test_account\",\"Namespace\":\"test_namespace\",\"Dims\":{\"key\":\"value\"},\"TS\":\"0001-01-01T00:00:00.000\"}:%d|g\n"
)

type testConn struct {
	buf bytes.Buffer
	err error
	net.Conn
}

func (c *testConn) Write(p []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.buf.Write(p)
}

func (c *testConn) Read(data []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.buf.Read(data)
}

func (c *testConn) Close() error {
	return c.err
}

func newTestClient() (*Statsd, error) {
	return &Statsd{
		conn:      &testConn{},
		account:   "test_account",
		namespace: "test_namespace",
		now: func() time.Time {
			time, _ := time.Parse("", "0001-01-01T00:00:00.000")
			return time
		},
	}, nil
}

func getOutput(c *Statsd) string {
	var buf bytes.Buffer
	io.Copy(&buf, c.conn)
	return buf.String()
}

func testOutput(t *testing.T, want string, f func(*Statsd)) {
	c, _ := newTestClient()
	defer c.Close()

	f(c)
	c.Close()

	got := getOutput(c)
	if got != want {
		t.Errorf("Invalid output, got:\n%q\nwant:\n%q", got, want)
	}
}

// TODO: Refactor these to smaller test
func TestEmitFloat(t *testing.T) {
	testOutput(t, fmt.Sprintf(testOutputF, float64(5)), func(c *Statsd) {
		c.EmitFloat(testKey, 5, map[string]string{"key": "value"})
	})
}

func TestEmitGauge(t *testing.T) {
	testOutput(t, fmt.Sprintf(testOutputG, 5), func(c *Statsd) {
		c.EmitGauge(testKey, 5, map[string]string{"key": "value"})
	})
}
