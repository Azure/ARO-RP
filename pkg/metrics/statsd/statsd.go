package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

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
)

// statsd implementation for https://genevamondocs.azurewebsites.net/collect/references/statsdref.html

// Now stubs time.Now
var Now = time.Now

var (
	defaultSocket = "/tmp/mdm_statsd.socket"
)

// Statsd defines internal statsd client
// It should be cloned, but not modified
type Statsd struct {
	account   string
	namespace string
	dims      map[string]string

	socket string
	conn   net.Conn
	env    env.Interface
	mu     sync.Mutex

	// for testing
	Muted bool
}

// New method to initialize udp connection
func New(ctx context.Context, log *logrus.Entry, _env env.Interface) (metrics.Interface, error) {
	// defaults- dev
	config := &Statsd{
		account:   os.Getenv("METRICS_ACCOUNT"),
		namespace: os.Getenv("METRICS_NAMESPACE"),
		dims: map[string]string{
			"location": _env.Location(),
		},
		socket: defaultSocket,
	}

	var err error
	config.conn, err = net.Dial("unix", config.socket)
	if err != nil {
		if _, ok := _env.(env.Dev); ok {
			log.Printf("Running in development, no metrics socket found %v", err)
			config.Muted = true
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
func (c *Statsd) EmitFloat(stat string, value float64, dims ...string) error {
	dimsNew := c.AppendDimensions(dims...)
	f := metric{
		Account:    c.account,
		Namespace:  c.namespace,
		Dims:       dimsNew,
		TS:         Now(),
		Metric:     stat,
		ValueFloat: to.Float64Ptr(float64(value)),
	}
	b, err := f.Marshal()
	if err != nil {
		return err
	}

	_, err = c.Send(b)
	if err != nil {
		return err
	}
	return nil
}

// EmitGauge records gauge information
func (c *Statsd) EmitGauge(stat string, value int64, dims ...string) error {
	dimsNew := c.AppendDimensions(dims...)
	g := metric{
		Account:    c.account,
		Namespace:  c.namespace,
		Dims:       dimsNew,
		TS:         Now(),
		Metric:     stat,
		ValueGauge: to.Int64Ptr(int64(value)),
	}
	b, err := g.Marshal()
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
	if c.Muted {
		return 0, nil
	}
	c.mu.Lock()
	bytesSent, err := c.conn.Write(data)
	c.mu.Unlock()
	// TODO: Add metric on how much bytes we sent
	return bytesSent, err
}

func (c *Statsd) AppendDimensions(dims ...string) map[string]string {
	if len(dims)%2 != 0 {
		panic("statsd: Tags only accepts an even number of arguments")
	}

	if len(dims) == 0 {
		return c.dims
	}

	newDims := make(map[string]string, (len(dims)/2 + len(c.dims)))
	for i := 0; i < len(dims)/2; i++ {
		newDims[dims[2*i]] = dims[2*i+1]
	}

	// append default dims
	for k, v := range c.dims {
		newDims[k] = v
	}
	return newDims
}
