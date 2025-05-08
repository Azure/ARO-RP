package golang

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type Runnable interface {
	Run()
}

// golang implements a goroutine which outputs standard Prometheus golang
// metrics to statsd.  TODO: I think new versions of the mdm agent can now poll
// Prometheus-compatible /metrics endpoints: if this is the case we can probably
// refactor our entire metrics stack to do things that way.
type golang struct {
	log *logrus.Entry
	m   metrics.Emitter
	r   *prometheus.Registry
}

func NewMetrics(log *logrus.Entry, m metrics.Emitter) (Runnable, error) {
	g := &golang{
		log: log,
		m:   m,
		r:   prometheus.NewRegistry(),
	}

	if err := g.r.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return nil, err
	}

	if err := g.r.Register(collectors.NewGoCollector()); err != nil {
		return nil, err
	}

	return g, nil
}

func (g *golang) Run() {
	defer recover.Panic(g.log)

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for range t.C {
		families, err := g.r.Gather()
		if err != nil {
			g.log.Error(err)
			continue
		}

		for _, family := range families {
			for _, metric := range family.Metric {
				m := map[string]string{}
				for _, label := range metric.Label {
					m[*label.Name] = *label.Value
				}

				switch {
				case metric.Counter != nil:
					g.m.EmitFloat(*family.Name, *metric.Counter.Value, m)
				case metric.Gauge != nil:
					g.m.EmitFloat(*family.Name, *metric.Gauge.Value, m)
				}
			}
		}
	}
}
