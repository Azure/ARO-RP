package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
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
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
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

	m := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

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

	clusterm := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("CLUSTER_MDM_ACCOUNT"), os.Getenv("CLUSTER_MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
	if err != nil {
		return err
	}

	if err := env.ValidateVars(envKeyVaultPrefix); err != nil {
		return err
	}
	keyVaultPrefix := os.Getenv(envKeyVaultPrefix)
	// TODO: should not be using the service keyvault here
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	if err := env.ValidateVars(envDatabaseAccountName); err != nil {
		return err
	}

	dbAccountName := os.Getenv(envDatabaseAccountName)
	clientOptions := &policy.ClientOptions{
		ClientOptions: _env.Environment().ManagedIdentityCredentialOptions().ClientOptions,
	}
	logrusEntry := log.WithField("component", "database")
	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, logrusEntry, msiToken, clientOptions, _env.SubscriptionID(), _env.ResourceGroup(), dbAccountName)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, &noop.Noop{}, aead, dbAccountName)
	if err != nil {
		return err
	}

	dbName, err := DBName(_env.IsLocalDevelopmentMode())
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

	mon := pkgmonitor.NewMonitor(log.WithField("component", "monitor"), dialer, dbMonitors, dbOpenShiftClusters, dbSubscriptions, m, clusterm, liveConfig, _env)

	return mon.Run(ctx)
}
