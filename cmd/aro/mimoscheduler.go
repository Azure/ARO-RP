package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/tracing"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/mimo/scheduler"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func mimoScheduler(ctx context.Context, _log *logrus.Entry) error {
	stop := make(chan struct{})

	_env, err := env.NewEnv(ctx, _log, env.SERVICE_MIMO_SCHEDULER)
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

	log := _env.Logger()

	m := statsd.New(ctx, _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))
	go m.Run(stop)

	g, err := golang.NewMetrics(log, m)
	if err != nil {
		return err
	}
	go g.Run()

	tracing.Register(azure.New(m))

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

	subscriptions, err := database.NewSubscriptions(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	manifests, err := database.NewMaintenanceManifests(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	schedules, err := database.NewMaintenanceSchedules(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	poolWorkers, err := database.NewPoolWorkers(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbg := database.NewDBGroup().
		WithOpenShiftClusters(clusters).
		WithSubscriptions(subscriptions).
		WithMaintenanceManifests(manifests).
		WithMaintenanceSchedules(schedules).
		WithPoolWorkers(poolWorkers)

	a := scheduler.NewService(_env, log, dbg, m)
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
