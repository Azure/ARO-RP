package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// statsd implementation for https://genevamondocs.azurewebsites.net/collect/references/statsdref.html
import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type statsd struct {
	log *logrus.Entry
	env env.Lite

	hostname  string
	account   string
	namespace string

	conn net.Conn
	ch   chan *metric

	now func() time.Time
}

// New returns a new metrics.Interface
func New(ctx context.Context, log *logrus.Entry, env env.Lite, account, namespace string) (metrics.Interface, error) {
	s := &statsd{
		log: log,
		env: env,

		account:   account,
		namespace: namespace,

		ch: make(chan *metric, 1024),

		now: time.Now,
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	s.hostname = hostname

	if s.account == "" {
		s.account = "*"
	}

	if s.namespace == "" {
		s.namespace = "*"
	}

	go s.run()

	return s, nil
}

// EmitFloat records float information
func (s *statsd) EmitFloat(m string, value float64, dims map[string]string) {
	s.emitMetric(&metric{
		metric:     m,
		dims:       dims,
		valueFloat: &value,
	})
}

// EmitGauge records gauge information
func (s *statsd) EmitGauge(m string, value int64, dims map[string]string) {
	s.emitMetric(&metric{
		metric:     m,
		dims:       dims,
		valueGauge: &value,
	})
}

func (s *statsd) emitMetric(m *metric) {
	m.account = s.account
	m.namespace = s.namespace
	if m.dims == nil {
		m.dims = map[string]string{}
	}
	m.dims["location"] = s.env.Location()
	m.dims["hostname"] = s.hostname
	m.ts = s.now()

	s.ch <- m
}

func (s *statsd) run() {
	defer recover.Panic(s.log)

	var lastLog time.Time

	for m := range s.ch {
		err := s.write(m)
		if err != nil &&
			s.now().After(lastLog.Add(time.Second)) {
			lastLog = s.now()
			s.log.Error(err)
		}
	}
}

func (s *statsd) dial() (err error) {
	metricsSocketPath := "/var/etw/mdm_statsd.socket"
	if s.env.Type() == env.Dev {
		metricsSocketPath = "mdm_statsd.socket"
	}

	s.conn, err = net.Dial("unix", metricsSocketPath)
	return
}

func (s *statsd) write(m *metric) (err error) {
	if s.now().After(m.ts.Add(time.Minute)) {
		return fmt.Errorf("discarding stale metric")
	}

	b, err := m.marshalStatsd()
	if err != nil {
		return
	}

	if s.conn == nil {
		err = s.dial()
		if err != nil {
			if s.env.Type() == env.Dev {
				err = nil
			}
			return
		}
	}

	defer func() {
		if err != nil && s.conn != nil {
			s.conn.Close()
			s.conn = nil
		}
	}()

	err = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return
	}

	_, err = s.conn.Write(b)
	return
}
