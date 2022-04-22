package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// statsd implementation for https://genevamondocs.azurewebsites.net/collect/references/statsdref.html
import (
	"context"
	"fmt"
	"net"
	"os"
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

	account   string
	namespace string

	conn net.Conn
	ch   chan *metric

	now func() time.Time
}

const statsdSocketEnv = "ARO_STATSD_SOCKET"

// New returns a new metrics.Interface
func New(ctx context.Context, log *logrus.Entry, env env.Core, account, namespace string) metrics.Emitter {
	s := &statsd{
		log: log,
		env: env,

		account:   account,
		namespace: namespace,

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
	const (
		malformed           = "malformed ENV variable ARO_STATSD_SOCKET. Expecting udp:<hostname>:<port> or unix:<path-to-socket> format. Got: %s"
		unsupportedProtocol = "unsupported protocol in ENV variable ARO_STATSD_SOCKET. Expecting  'udp:' or 'unix:'. Got: %s in %s"
		invalidUDPAddress   = "invalid UDP address in ENV variable ARO_STATSD_SOCKET %s. Error: %s "
	)

	// Verify protocol:connectionstring format
	parameters := strings.SplitN(env, ":", 2)
	if len(parameters) != 2 {
		return "", "", fmt.Errorf(malformed, env)
	}
	protocol := parameters[0]
	connectionstring := parameters[1]

	//Verify supported protocol provided
	if protocol != "udp" && protocol != "unix" {
		return "", "", fmt.Errorf(unsupportedProtocol, protocol, env)
	}

	//UDP address check, no such (meaningful) thing for unix:
	if protocol == "udp" {
		_, err := net.ResolveUDPAddr(protocol, connectionstring)
		if err != nil {
			return "", "", fmt.Errorf(invalidUDPAddress, env, err)
		}
	}

	return protocol, connectionstring, nil
}

func (s *statsd) getDefaultSocketValues() (string, string) {
	protocol := "unix"
	connectionstring := "/var/etw/mdm_statsd.socket"

	if s.env.IsLocalDevelopmentMode() {
		connectionstring = "mdm_statsd.socket"
	}

	return protocol, connectionstring
}

func (s *statsd) getConnectionDetails() (string, string, error) {
	// allow the socket connection to be overriden via ENV Variable
	socketEnv, isset := os.LookupEnv(statsdSocketEnv)
	if !isset { //original behaviour
		protocol, connectionstring := s.getDefaultSocketValues()
		return protocol, connectionstring, nil
	}

	return s.parseSocketEnv(socketEnv)
}

func (s *statsd) dial() (err error) {
	protocol, connectionstring, err := s.getConnectionDetails()
	if err != nil {
		return
	}

	s.conn, err = net.Dial(protocol, connectionstring)

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
