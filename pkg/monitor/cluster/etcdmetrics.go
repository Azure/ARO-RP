package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/Azure/ARO-RP/pkg/proxy/prometheus"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
)

func (mon *Monitor) emitEtcdMetrics(ctx context.Context) error {
	var etcdObjectCount int64 = -1

	prom := prometheus.NewPrometheusProxyWithRestConfig(mon.log, mon.restconfig)

	promClient, err := api.NewClient(api.Config{Address: "http://prometheus-k8s-0:9090", RoundTripper: roundtripper.RoundTripperFunc(prom.RoundTripper)})
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		v1api := v1.NewAPI(promClient)
		result, warnings, err := v1api.Query(ctx, "round(sum(instance:etcd_object_counts:sum)/3)", time.Now())
		if err != nil {
			mon.log.Errorf("Error querying Prometheus: %v\n", err)
			break
		}

		if len(warnings) > 0 {
			mon.log.Info(warnings)
		}

		objs := result.(model.Vector)
		if len(objs) != 1 {
			// idk what's going on here, bail
			break
		}
		etcdObjectCount = int64(objs[0].Value)
	}
	if err != nil {
		return err
	}

	if etcdObjectCount != -1 {
		mon.emitGauge("prometheus.etcd.objects.count", int64(etcdObjectCount), nil)
	}

	return nil
}
