package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// statsd implementation for https://genevamondocs.azurewebsites.net/collect/references/statsdref.html
import (
	"context"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/go-autorest/tracing"
	"github.com/sirupsen/logrus"
	k8smetrics "k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	statsdazure "github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	statsdk8s "github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
)

const defaultSocket = "mdm_statsd.socket"

type statsd struct {
	env  env.Interface
	conn io.WriteCloser
	mu   sync.Mutex

	now func() time.Time

	hostname string
}

// New returns a new metrics.Interface
func New(ctx context.Context, log *logrus.Entry, _env env.Interface) (metrics.Interface, error) {
	s := &statsd{
		env: _env,
		now: time.Now,
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	s.hostname = hostname

	s.conn, err = net.Dial("unix", defaultSocket)
	if _, ok := _env.(env.Dev); ok &&
		err != nil &&
		strings.HasSuffix(err.Error(), "connect: no such file or directory") {
		log.Printf("%s does not exist; not outputting metrics", defaultSocket)
		return &noop.Noop{}, nil
	}
	if err != nil {
		return nil, err
	}

	// register azure client tracer
	tracing.Register(statsdazure.New(s))

	// register k8s client tracer
	k8smetrics.Register(statsdk8s.NewLatency(s), statsdk8s.NewResult(s))

	return s, nil
}

// Close closes the connection
func (s *statsd) Close() error {
	return s.conn.Close()
}

// EmitFloat records float information
func (s *statsd) EmitFloat(m string, value float64, dims map[string]string) error {
	return s.emitMetric(metric{
		metric:     m,
		dims:       dims,
		valueFloat: &value,
	})
}

// EmitGauge records gauge information
func (s *statsd) EmitGauge(m string, value int64, dims map[string]string) error {
	return s.emitMetric(metric{
		metric:     m,
		dims:       dims,
		valueGauge: &value,
	})
}

func (s *statsd) emitMetric(m metric) error {
	m.account = "*"
	m.namespace = "*"
	if m.dims == nil {
		m.dims = map[string]string{}
	}
	m.dims["location"] = s.env.Location()
	m.dims["hostname"] = s.hostname
	m.ts = s.now()

	b, err := m.marshalStatsd()
	if err != nil {
		return err
	}

	_, err = s.Write(b)
	return err
}

// Send data to statsd daemon
func (s *statsd) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Write(data)
}
