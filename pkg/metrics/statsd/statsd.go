package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// statsd implementation for https://genevamondocs.azurewebsites.net/collect/references/statsdref.html
import (
	"context"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
)

var (
	defaultSocket = "mdm_statsd.socket"
)

// Statsd defines internal statsd client
// It should be cloned, but not modified
type Statsd struct {
	account   string
	namespace string

	conn net.Conn
	env  env.Interface
	mu   sync.Mutex

	now func() time.Time
}

// New method to initialize udp connection
func New(ctx context.Context, log *logrus.Entry, _env env.Interface) (metrics.Interface, error) {
	// defaults- dev
	config := &Statsd{
		account:   os.Getenv("METRICS_ACCOUNT"),
		namespace: os.Getenv("METRICS_NAMESPACE"),
		now:       time.Now,
	}

	var err error
	config.conn, err = net.Dial("unix", defaultSocket)
	if err != nil {
		if _, ok := _env.(env.Dev); ok {
			log.Printf("Running in development, no metrics socket found %v", err)
			return &noop.Noop{}, nil
		} else if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// Close closes the connection
func (c *Statsd) Close() {
	c.conn.Close()
}

// EmitFloat records float information
func (c *Statsd) EmitFloat(stat string, value float64, dims map[string]string) error {
	return c.emitMetric(metric{
		Metric:     stat,
		Dims:       dims,
		ValueFloat: to.Float64Ptr(float64(value)),
	})
}

// EmitGauge records gauge information
func (c *Statsd) EmitGauge(stat string, value int64, dims map[string]string) error {
	return c.emitMetric(metric{
		Metric:     stat,
		Dims:       dims,
		ValueGauge: to.Int64Ptr(int64(value)),
	})
}

func (c *Statsd) emitMetric(m metric) error {
	m.Account = c.account
	m.Namespace = c.namespace
	m.TS = c.now()

	b, err := m.MarshalStatsd()
	if err != nil {
		return err
	}
	_, err = c.Send(b)
	if err != nil {
		return err
	}
	return nil
}

// Send data to statsd daemon
func (c *Statsd) Send(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Write(data)
}
