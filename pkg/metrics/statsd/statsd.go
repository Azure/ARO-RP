package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// statsd implementation for https://genevamondocs.azurewebsites.net/collect/references/statsdref.html
import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type statsd struct {
	log *logrus.Entry
	env env.Core

	account      string
	namespace    string
	mdmsocketEnv string

	conn net.Conn
	ch   chan *metric

	now func() time.Time
}

// New returns a new metrics.Interface
func New(ctx context.Context, log *logrus.Entry, env env.Core, account, namespace string, mdmsocketEnv string) metrics.Emitter {
	s := &statsd{
		log: log,
		env: env,

		account:      account,
		namespace:    namespace,
		mdmsocketEnv: mdmsocketEnv,

		ch: make(chan *metric, 1024),

		now: time.Now,
	}

	if s.account == "" {
		s.account = "*"
	}

	if s.namespace == "" {
		s.namespace = "*"
	}

	go s.run()

	return s
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
	m.dims["hostname"] = s.env.Hostname()
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

func (s *statsd) parseSocketEnv(env string) (string, string, error) {
	// Verify network:address format
	parameters := strings.SplitN(env, ":", 2)
	if len(parameters) != 2 {
		return "", "", fmt.Errorf("malformed definition for the mdm statds socket. Expecting udp:<hostname>:<port> or unix:<path-to-socket> format. Got: %q", env)
	}
	network := strings.ToLower(parameters[0])
	address := parameters[1]
	return network, address, nil
}

func (s *statsd) validateSocketDefinition(network string, address string) (bool, error) {
	//Verify supported protocol provided. TCP might just work as well, but this was never tested
	if network != "udp" && network != "unix" {
		return false, fmt.Errorf("unsupported protocol for the mdm statds socket. Expecting  'udp:' or 'unix:'. Got: %q", network)
	}

	return true, nil
}

func (s *statsd) getDefaultSocketValues() (string, string) {
	network := "unix"
	address := "/var/etw/mdm_statsd.socket"

	if s.env.IsLocalDevelopmentMode() {
		address = "mdm_statsd.socket"
	}

	return network, address
}

func (s *statsd) getConnectionDetails() (string, string, error) {
	// allow the default socket connection to be overriden
	if s.mdmsocketEnv == "" { //original behaviour
		network, address := s.getDefaultSocketValues()
		return network, address, nil
	}

	network, address, err := s.parseSocketEnv(s.mdmsocketEnv)
	if err != nil {
		return "", "", err
	}

	ok, err := s.validateSocketDefinition(network, address)
	if !ok {
		return "", "", err
	}

	return network, address, nil
}

func (s *statsd) dial() (err error) {
	network, address, err := s.getConnectionDetails()
	if err != nil {
		return
	}

	s.conn, err = net.Dial(network, address)

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
			if s.env.IsLocalDevelopmentMode() {
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
