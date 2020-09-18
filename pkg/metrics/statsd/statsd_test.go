package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestEmitGauge(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().Location().AnyTimes().Return("eastus")

	c1, c2 := net.Pipe()

	s := &statsd{
		env: env,

		account:   "*",
		namespace: "*",

		conn: c1,
		ch:   make(chan *metric),

		now: func() time.Time { return time.Time{} },
	}

	go s.run()

	s.EmitGauge("tests.test_key", 42, map[string]string{"key": "value"})

	m, err := bufio.NewReader(c2).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if m != `{"Metric":"tests.test_key","Account":"*","Namespace":"*","Dims":{"hostname":"","key":"value","location":"eastus"},"TS":"0001-01-01T00:00:00.000"}:42|g`+"\n" {
		t.Error(m)
	}
}

func TestEmitFloat(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().Location().AnyTimes().Return("eastus")

	c1, c2 := net.Pipe()

	s := &statsd{
		env: env,

		account:   "*",
		namespace: "*",

		conn: c1,
		ch:   make(chan *metric),

		now: func() time.Time { return time.Time{} },
	}

	go s.run()

	s.EmitFloat("tests.test_key", 5, map[string]string{"key": "value"})

	m, err := bufio.NewReader(c2).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if m != `{"Metric":"tests.test_key","Account":"*","Namespace":"*","Dims":{"hostname":"","key":"value","location":"eastus"},"TS":"0001-01-01T00:00:00.000"}:5.000000|f`+"\n" {
		t.Error(m)
	}
}
