package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	kmetrics "k8s.io/client-go/tools/metrics"

	"github.com/Azure/go-autorest/tracing"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
	pkgmonitor "github.com/Azure/ARO-RP/pkg/monitor"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func monitor(ctx context.Context, _log *logrus.Entry) error {
	_env, err := env.NewEnv(ctx, _log, env.SERVICE_MONITOR)
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

	m := statsd.New(ctx, _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(_env.LoggerForComponent("metrics"), m)
	if err != nil {
		return err
	}

	go g.Run()

	tracing.Register(azure.New(m))
	kmetrics.Register(kmetrics.RegisterOpts{
		RequestResult:  k8s.NewResult(m),
		RequestLatency: k8s.NewLatency(m),
	})

	clusterm := statsd.NewMetricsForCluster(ctx, _env, os.Getenv("CLUSTER_MDM_ACCOUNT"), os.Getenv("CLUSTER_MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	aead, err := encryption.NewAEADWithCore(ctx, _env, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClientFromEnv(ctx, _env, m, aead)
	if err != nil {
		return err
	}

	dbName, err := env.DBName(_env)
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

	dbg := database.NewDBGroup().WithOpenShiftClusters(dbOpenShiftClusters).
		WithSubscriptions(dbSubscriptions).
		WithMonitors(dbMonitors)

	dialer, err := proxy.NewDialer(_env.IsLocalDevelopmentMode(), _env.LoggerForComponent("dialer"))
	if err != nil {
		return err
	}

	mon := pkgmonitor.NewMonitor(_env.LoggerForComponent("monitor"), dialer, dbg, m, clusterm, _env)

	return mon.Run(ctx)
}
