package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

type Monitor struct {
	env env.Interface
	log *logrus.Entry

	oc   *api.OpenShiftCluster
	dims map[string]string

	cli kubernetes.Interface
	m   metrics.Interface
}

func NewMonitor(env env.Interface, log *logrus.Entry, oc *api.OpenShiftCluster, m metrics.Interface) (*Monitor, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	dims := map[string]string{
		"resourceId":     oc.ID,
		"subscriptionId": r.SubscriptionID,
		"resourceGroup":  r.ResourceGroup,
		"resourceName":   r.ResourceName,
	}

	restConfig, err := restconfig.RestConfig(env, oc)
	if err != nil {
		return nil, err
	}

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &Monitor{
		env: env,
		log: log,

		oc:   oc,
		dims: dims,

		cli: cli,
		m:   m,
	}, nil
}

// Monitor checks the API server health of a cluster
func (mon *Monitor) Monitor(ctx context.Context) error {
	mon.log.Debug("monitoring")

	err := mon.emitClusterVersion(ctx)
	if err != nil {
		return err
	}

	// If API is not returning 200, don't need to run the next checks
	statusCode, err := mon.emitAPIServerHealthzCode(ctx)
	if err != nil || statusCode != http.StatusOK {
		return err
	}

	return mon.emitPrometheusAlerts(ctx)
}

func (mon *Monitor) emitFloat(m string, value float64, dims map[string]string) {
	for k, v := range mon.dims {
		dims[k] = v
	}
	mon.m.EmitFloat(m, value, dims)
}

func (mon *Monitor) emitGauge(m string, value int64, dims map[string]string) {
	for k, v := range mon.dims {
		dims[k] = v
	}
	mon.m.EmitGauge(m, value, dims)
}
