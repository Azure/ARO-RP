package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/go-autorest/tracing"
	"github.com/sirupsen/logrus"
	kmetrics "k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
	pkgmonitor "github.com/Azure/ARO-RP/pkg/monitor"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

func monitor(ctx context.Context, log *logrus.Entry) error {
	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	if !_env.IsLocalDevelopmentMode() {
		for _, key := range []string{
			"CLUSTER_MDM_ACCOUNT",
			"CLUSTER_MDM_NAMESPACE",
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
		} {
			if _, found := os.LookupEnv(key); !found {
				return fmt.Errorf("environment variable %q unset", key)
			}
		}
	}

	m := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"))

	tracing.Register(azure.New(m))
	kmetrics.Register(kmetrics.RegisterOpts{
		RequestResult:  k8s.NewResult(m),
		RequestLatency: k8s.NewLatency(m),
	})

	clusterm := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("CLUSTER_MDM_ACCOUNT"), os.Getenv("CLUSTER_MDM_NAMESPACE"))

	msiAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return err
	}

	// TODO: should not be using the service keyvault here
	serviceKeyvaultURI, err := keyvault.URI(_env, env.ServiceKeyvaultSuffix)
	if err != nil {
		return err
	}

	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, _env, msiAuthorizer)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, &noop.Noop{}, aead)
	if err != nil {
		return err
	}

	dbMonitors, err := database.NewMonitors(ctx, _env.IsLocalDevelopmentMode(), dbc)
	if err != nil {
		return err
	}

	dbOpenShiftClusters, err := database.NewOpenShiftClusters(ctx, _env.IsLocalDevelopmentMode(), dbc)
	if err != nil {
		return err
	}

	dbSubscriptions, err := database.NewSubscriptions(ctx, _env.IsLocalDevelopmentMode(), dbc)
	if err != nil {
		return err
	}

	dialer, err := proxy.NewDialer(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	mon := pkgmonitor.NewMonitor(log.WithField("component", "monitor"), dialer, dbMonitors, dbOpenShiftClusters, dbSubscriptions, m, clusterm)

	return mon.Run(ctx)
}
