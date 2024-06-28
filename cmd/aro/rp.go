package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Azure/go-autorest/tracing"
	"github.com/sirupsen/logrus"
	kmetrics "k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/api/admin"
	_ "github.com/Azure/ARO-RP/pkg/api/v20191231preview"
	_ "github.com/Azure/ARO-RP/pkg/api/v20200430"
	_ "github.com/Azure/ARO-RP/pkg/api/v20210901preview"
	_ "github.com/Azure/ARO-RP/pkg/api/v20220401"
	_ "github.com/Azure/ARO-RP/pkg/api/v20220904"
	_ "github.com/Azure/ARO-RP/pkg/api/v20230401"
	_ "github.com/Azure/ARO-RP/pkg/api/v20230701preview"
	_ "github.com/Azure/ARO-RP/pkg/api/v20230904"
	_ "github.com/Azure/ARO-RP/pkg/api/v20231122"
	_ "github.com/Azure/ARO-RP/pkg/api/v20240812preview"
	"github.com/Azure/ARO-RP/pkg/backend"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
	"github.com/Azure/ARO-RP/pkg/util/clusterdata"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func rp(ctx context.Context, log, audit *logrus.Entry) error {
	stop := make(chan struct{})

	_env, err := env.NewEnv(ctx, log, env.COMPONENT_RP)
	if err != nil {
		return err
	}

	var keys []string
	if _env.IsLocalDevelopmentMode() {
		keys = []string{
			"PULL_SECRET",
		}
	} else {
		keys = []string{
			"ACR_RESOURCE_ID",
			"ADMIN_API_CLIENT_CERT_COMMON_NAME",
			"CLUSTER_MDM_ACCOUNT",
			"CLUSTER_MDM_NAMESPACE",
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
		}

		if _, found := os.LookupEnv("PULL_SECRET"); found {
			return fmt.Errorf(`environment variable "PULL_SECRET" set`)
		}
	}

	if err = env.ValidateVars(keys...); err != nil {
		return err
	}

	err = _env.InitializeAuthorizers()
	if err != nil {
		return err
	}

	metrics := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(log.WithField("component", "metrics"), metrics)
	if err != nil {
		return err
	}

	go g.Run()

	tracing.Register(azure.New(metrics))
	kmetrics.Register(kmetrics.RegisterOpts{
		RequestResult:  k8s.NewResult(metrics),
		RequestLatency: k8s.NewLatency(metrics),
	})

	clusterm := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("CLUSTER_MDM_ACCOUNT"), os.Getenv("CLUSTER_MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return err
	}

	aead, err := encryption.NewMulti(ctx, _env.ServiceKeyvault(), env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	if err := env.ValidateVars(envDatabaseAccountName); err != nil {
		return err
	}
	dbAccountName := os.Getenv(envDatabaseAccountName)

	logrusEntry := log.WithField("component", "database")
	scope := []string{fmt.Sprintf("https://%s.%s", dbAccountName, _env.Environment().CosmosDBDNSSuffixScope)}
	dbAuthorizer, err := database.NewTokenAuthorizer(ctx, logrusEntry, msiToken, dbAccountName, scope)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, metrics, aead, dbAccountName)
	if err != nil {
		return err
	}

	dbName, err := DBName(env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	dbAsyncOperations, err := database.NewAsyncOperations(ctx, _env.IsLocalDevelopmentMode(), dbc, dbName)
	if err != nil {
		return err
	}

	dbClusterManagerConfiguration, err := database.NewClusterManagerConfigurations(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbBilling, err := database.NewBilling(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbGateway, err := database.NewGateway(ctx, dbc, dbName)
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

	dbOpenShiftVersions, err := database.NewOpenShiftVersions(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbPlatformWorkloadIdentityRoleSets, err := database.NewPlatformWorkloadIdentityRoleSets(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	go database.EmitMetrics(ctx, log, dbOpenShiftClusters, metrics)

	feAead, err := encryption.NewMulti(ctx, _env.ServiceKeyvault(), env.FrontendEncryptionSecretV2Name, env.FrontendEncryptionSecretName)
	if err != nil {
		return err
	}
	hiveClusterManager, err := hive.NewFromEnv(ctx, log, _env)
	if err != nil {
		return err
	}
	f, err := frontend.NewFrontend(ctx, audit, log.WithField("component", "frontend"), _env, dbAsyncOperations, dbClusterManagerConfiguration, dbOpenShiftClusters, dbSubscriptions, dbOpenShiftVersions, dbPlatformWorkloadIdentityRoleSets, api.APIs, metrics, clusterm, feAead, hiveClusterManager, adminactions.NewKubeActions, adminactions.NewAzureActions, clusterdata.NewParallelEnricher(metrics, _env))
	if err != nil {
		return err
	}

	b, err := backend.NewBackend(ctx, log.WithField("component", "backend"), _env, dbAsyncOperations, dbBilling, dbGateway, dbOpenShiftClusters, dbSubscriptions, dbOpenShiftVersions, dbPlatformWorkloadIdentityRoleSets, aead, metrics)
	if err != nil {
		return err
	}

	// This part of the code orchestrates shutdown sequence. When sigterm is
	// received, it will trigger backend to stop accepting new documents and
	// finish old ones. Frontend will stop advertising itself to the loadbalancer.
	// When shutdown completes for frontend and backend "/healthz" endpoint
	// will go dark and external observer will know that shutdown sequence is finished
	sigterm := make(chan os.Signal, 1)
	doneF := make(chan struct{})
	doneB := make(chan struct{})
	signal.Notify(sigterm, syscall.SIGTERM)

	log.Print("listening")
	go b.Run(ctx, stop, doneB)
	go f.Run(ctx, stop, doneF)

	<-sigterm
	log.Print("received SIGTERM")
	close(stop)
	<-doneB
	<-doneF

	return nil
}
