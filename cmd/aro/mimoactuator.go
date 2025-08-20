package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/mimo/actuator"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func mimoActuator(ctx context.Context, _log *logrus.Entry) error {
	stop := make(chan struct{})

	_env, err := env.NewEnv(ctx, _log, env.COMPONENT_MIMO_ACTUATOR)
	if err != nil {
		return err
	}

	keys := []string{}
	if !_env.IsLocalDevelopmentMode() {
		keys = []string{
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
		}
	}

	if err = env.ValidateVars(keys...); err != nil {
		return err
	}

	m := statsd.New(ctx, _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(_env.Logger(), m)
	if err != nil {
		return err
	}
	go g.Run()

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

	go database.EmitMIMOMetrics(ctx, _env.LoggerForComponent("metrics"), manifests, m)

	dialer, err := proxy.NewDialer(_env.IsLocalDevelopmentMode(), _env.LoggerForComponent("dialer"))
	if err != nil {
		return err
	}

	a := actuator.NewService(_env, _env.Logger(), dialer, dbg, m)
	a.SetMaintenanceTasks(tasks.DEFAULT_MAINTENANCE_TASKS)

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
