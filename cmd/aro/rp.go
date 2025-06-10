package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	kmetrics "k8s.io/client-go/tools/metrics"

	"github.com/Azure/go-autorest/tracing"

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
	_ "github.com/Azure/ARO-RP/pkg/api/v20250725"
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
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
)

func rp(ctx context.Context, log, auditLog *logrus.Entry) error {
	stop := make(chan struct{})

	_env, err := env.NewEnv(ctx, log, env.COMPONENT_RP)
	if err != nil {
		return err
	}

	var keys []string
	if _env.IsLocalDevelopmentMode() {
		keys = []string{
			"PULL_SECRET",
			env.OIDCStorageAccountName,
		}
	} else {
		keys = []string{
			"ACR_RESOURCE_ID",
			"ADMIN_API_CLIENT_CERT_COMMON_NAME",
			"CLUSTER_MDM_ACCOUNT",
			"CLUSTER_MDM_NAMESPACE",
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
			"MSI_RP_ENDPOINT",
			env.OIDCStorageAccountName,
		}

		if _, found := os.LookupEnv("PULL_SECRET"); found {
			return fmt.Errorf(`environment variable "PULL_SECRET" set`)
		}
	}

	if !_env.FeatureIsSet(env.FeatureRequireOIDCStorageWebEndpoint) {
		if err := env.ValidateVars(env.OIDCAFDEndpoint); err != nil {
			return err
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

	aead, err := encryption.NewAEADWithCore(ctx, _env, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClientFromEnv(ctx, _env, log, metrics, aead)
	if err != nil {
		return err
	}

	dbName, err := env.DBName(_env)
	if err != nil {
		return err
	}

	dbAsyncOperations, err := database.NewAsyncOperations(ctx, _env.IsLocalDevelopmentMode(), dbc, dbName)
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

	// Note: When handling DB operations don't delete records but set TTL on them otherwise if we're leveraging change feeds, it will break.
	dbPlatformWorkloadIdentityRoleSets, err := database.NewPlatformWorkloadIdentityRoleSets(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	go database.EmitOpenShiftClustersMetrics(ctx, log, dbOpenShiftClusters, metrics)

	feAead, err := encryption.NewMulti(ctx, _env.ServiceKeyvault(), env.FrontendEncryptionSecretV2Name, env.FrontendEncryptionSecretName)
	if err != nil {
		return err
	}
	hiveClusterManager, err := hive.NewFromEnvCLusterManager(ctx, log, _env)
	if err != nil {
		return err
	}

	hiveSyncSetManager, err := hive.NewFromEnvSyncSetManager(ctx, log, _env)
	if err != nil {
		return err
	}

	dbg := database.NewDBGroup().WithAsyncOperations(dbAsyncOperations).
		WithBilling(dbBilling).
		WithOpenShiftClusters(dbOpenShiftClusters).
		WithOpenShiftVersions(dbOpenShiftVersions).
		WithPlatformWorkloadIdentityRoleSets(dbPlatformWorkloadIdentityRoleSets).
		WithSubscriptions(dbSubscriptions)

	// MIMO only activated in development for now
	if _env.IsLocalDevelopmentMode() {
		dbMaintenanceManifests, err := database.NewMaintenanceManifests(ctx, dbc, dbName)
		if err != nil {
			return err
		}
		dbg.WithMaintenanceManifests(dbMaintenanceManifests)
	}

	size, err := _env.OtelAuditQueueSize()
	if err != nil {
		return err
	}

	outelAuditClient, err := audit.NewOtelAuditClient(size, _env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	f, err := frontend.NewFrontend(ctx, auditLog, log.WithField("component", "frontend"), outelAuditClient, _env, dbg, api.APIs, metrics, clusterm, feAead, hiveClusterManager, hiveSyncSetManager, adminactions.NewKubeActions, adminactions.NewAzureActions, adminactions.NewAppLensActions, clusterdata.NewParallelEnricher(metrics, _env))
	if err != nil {
		return err
	}

	b, err := backend.NewBackend(log.WithField("component", "backend"), _env, dbAsyncOperations, dbBilling, dbGateway, dbOpenShiftClusters, dbSubscriptions, dbOpenShiftVersions, dbPlatformWorkloadIdentityRoleSets, aead, metrics)
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
