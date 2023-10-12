package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/tracing"
	"github.com/sirupsen/logrus"
	kmetrics "k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
	pkgmonitor "github.com/Azure/ARO-RP/pkg/monitor"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/service"
)

func monitor(ctx context.Context, log *logrus.Entry) error {
	_env, err := env.NewEnv(ctx, log, env.COMPONENT_MONITOR)
	if err != nil {
		return err
	}

	if !_env.IsLocalDevelopmentMode() {
		err := env.ValidateVars(
			"CLUSTER_MDM_ACCOUNT",
			"CLUSTER_MDM_NAMESPACE",
			"MDM_ACCOUNT",
			"MDM_NAMESPACE")

		if err != nil {
			return err
		}
	}

	if err := env.ValidateVars(
		service.KeyVaultPrefix,
		service.DatabaseAccountName,
	); err != nil {
		return err
	}

	m := statsd.NewFromEnv(ctx, log.WithField("component", "metrics"), _env)

	g, err := golang.NewMetrics(log.WithField("component", "metrics"), m)
	if err != nil {
		return err
	}

	go g.Run()

	tracing.Register(azure.New(m))
	kmetrics.Register(kmetrics.RegisterOpts{
		RequestResult:  k8s.NewResult(m),
		RequestLatency: k8s.NewLatency(m),
	})

	clusterm := statsd.NewFromEnv(ctx, log.WithField("component", "metrics"), _env, "CLUSTER")

	dbc, err := service.NewDatabase(ctx, _env, log, &noop.Noop{}, service.DB_ALWAYS_MASTERKEY, true)
	if err != nil {
		return err
	}

	dbName, err := service.DBName(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}
	dbMonitors, err := database.NewMonitors(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbOpenShiftClusters, err := database.NewOpenShiftClusters(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbSubscriptions, err := database.NewSubscriptions(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dialer, err := proxy.NewDialer(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	liveConfig, err := _env.NewLiveConfigManager(ctx)
	if err != nil {
		return err
	}

	mon := pkgmonitor.NewMonitor(_env.Logger(), dialer, dbMonitors, dbOpenShiftClusters, dbSubscriptions, m, clusterm, liveConfig, _env)

	return mon.Run(ctx)
}
