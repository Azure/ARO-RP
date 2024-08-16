package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/mimo/actuator"
	"github.com/Azure/ARO-RP/pkg/mimo/sets"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/service"
)

func mimoActuator(ctx context.Context, log *logrus.Entry) error {
	stop := make(chan struct{})

	_env, err := env.NewEnv(ctx, log, env.COMPONENT_MIMO_ACTUATOR)
	if err != nil {
		return err
	}

	var keys []string
	if _env.IsLocalDevelopmentMode() {
		keys = []string{}
	} else {
		keys = []string{
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
		}
	}

	if err = env.ValidateVars(keys...); err != nil {
		return err
	}

	m := statsd.New(ctx, log.WithField("component", "actuator"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(_env.Logger(), m)
	if err != nil {
		return err
	}
	go g.Run()

	dbc, err := service.NewDatabase(ctx, _env, log, m, false)
	if err != nil {
		return err
	}

	dbName, err := service.DBName(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	clusters, err := database.NewOpenShiftClusters(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	manifests, err := database.NewMaintenanceManifests(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbg := database.NewDBGroup().
		WithOpenShiftClusters(clusters).
		WithMaintenanceManifests(manifests)

	dialer, err := proxy.NewDialer(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	a := actuator.NewService(_env, _env.Logger(), dialer, dbg, m)
	a.SetMaintenanceSets(sets.DEFAULT_MAINTENANCE_SETS)

	sigterm := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(sigterm, syscall.SIGTERM)

	go a.Run(ctx, stop, done)

	<-sigterm
	log.Print("received SIGTERM")
	close(stop)
	<-done

	return nil
}
