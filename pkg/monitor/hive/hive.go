package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	"github.com/Azure/ARO-RP/pkg/monitor/emitter"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

var _ monitoring.Monitor = (*Monitor)(nil)

type collectorFunc func(context.Context) error

type Monitor struct {
	collectors []collectorFunc

	log *logrus.Entry

	hourlyRun bool
	oc        *api.OpenShiftCluster
	dims      map[string]string

	m metrics.Emitter

	hiveClusterManager hive.ClusterManager
}

func NewHiveMonitor(log *logrus.Entry, oc *api.OpenShiftCluster, m metrics.Emitter, hourlyRun bool, hiveClusterManager hive.ClusterManager) (*Monitor, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	dims := map[string]string{
		dimension.ResourceID:           oc.ID,
		dimension.SubscriptionID:       r.SubscriptionID,
		dimension.ClusterResourceGroup: r.ResourceGroup,
		dimension.ResourceName:         r.ResourceName,
	}

	mon := &Monitor{
		log: log,

		hourlyRun: hourlyRun,
		oc:        oc,
		dims:      dims,

		m: m,

		hiveClusterManager: hiveClusterManager,
	}
	mon.collectors = []collectorFunc{
		mon.emitHiveRegistrationStatus,
		mon.emitClusterSync,
	}

	return mon, nil
}

func (mon *Monitor) runCollector(ctx context.Context, f func(context.Context) error) (err error) {
	collectorName := steps.ShortName(f)
	mon.log.Debugf("running %s", collectorName)

	// If the collector panics we should return the error (so that it bubbles
	// up) but not prevent any other collector from running.
	defer func() {
		if e := recover(); e != nil {
			err = &failureToRunHiveCollector{collectorName: collectorName, inner: &collectorPanic{panicValue: e}}
			mon.emitHiveCollectorError(collectorName)
		}
	}()

	innerErr := f(ctx)
	if innerErr != nil {
		// emit metrics collection failures and collect the err, but
		// don't stop running other metric collections
		mon.emitHiveCollectorError(collectorName)
		return &failureToRunHiveCollector{collectorName: collectorName, inner: innerErr}
	}
	return nil
}

// Monitor checks the health of Hive resources associated with a cluster
func (mon *Monitor) Monitor(ctx context.Context) error {
	now := time.Now()

	mon.log.Debug("hive monitoring")

	errs := []error{}
	for _, f := range mon.collectors {
		innerErr := mon.runCollector(ctx, f)
		if innerErr != nil {
			errs = append(errs, innerErr)
		}
	}

	// emit a metric with how long we took when we have no errors
	if len(errs) == 0 {
		mon.emitFloat("monitor.hive.duration", time.Since(now).Seconds(), map[string]string{})
	}

	return errors.Join(errs...)
}

func (mon *Monitor) emitHiveCollectorError(collectorName string) {
	emitter.EmitGauge(mon.m, "monitor.hive.collector.error", 1, mon.dims, map[string]string{"collector": collectorName})
}

func (mon *Monitor) emitGauge(m string, value int64, dims map[string]string) {
	emitter.EmitGauge(mon.m, m, value, mon.dims, dims)
}

func (mon *Monitor) emitFloat(m string, value float64, dims map[string]string) {
	emitter.EmitFloat(mon.m, m, value, mon.dims, dims)
}
