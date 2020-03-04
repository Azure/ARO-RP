package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

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

	oc *api.OpenShiftCluster

	cli kubernetes.Interface
	m   metrics.Interface
}

func NewMonitor(ctx context.Context, env env.Interface, log *logrus.Entry, oc *api.OpenShiftCluster, m metrics.Interface) (*Monitor, error) {
	restConfig, err := restconfig.RestConfig(ctx, env, oc)
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

		oc: oc,

		cli: cli,
		m:   m,
	}, nil
}

// Monitor checks the API server health of a cluster
func (mon *Monitor) Monitor(ctx context.Context) error {
	mon.log.Debug("monitoring")

	// If API is not returning 200, don't need to run the next checks
	statusCode, err := mon.emitAPIServerHealthzCode(ctx)
	if err != nil || statusCode != http.StatusOK {
		return err
	}

	return mon.emitPrometheusAlerts(ctx)
}
