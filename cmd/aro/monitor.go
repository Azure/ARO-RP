package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/go-autorest/tracing"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
	pkgmonitor "github.com/Azure/ARO-RP/pkg/monitor"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

func monitor(ctx context.Context, log *logrus.Entry) error {
	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	if _env.DeploymentMode() != deployment.Development {
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
	metrics.Register(metrics.RegisterOpts{
		RequestResult:  k8s.NewResult(m),
		RequestLatency: k8s.NewLatency(m),
	})

	clusterm := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("CLUSTER_MDM_ACCOUNT"), os.Getenv("CLUSTER_MDM_NAMESPACE"))

	rpKVAuthorizer, err := _env.NewRPAuthorizer(_env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return err
	}

	// TODO: should not be using the service keyvault here
	serviceKeyvaultURI, err := keyvault.URI(_env, generator.ServiceKeyvaultSuffix)
	if err != nil {
		return err
	}

	serviceKeyvault := keyvault.NewManager(rpKVAuthorizer, serviceKeyvaultURI)

	key, err := serviceKeyvault.GetBase64Secret(ctx, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	aead, err := encryption.NewXChaCha20Poly1305(ctx, key)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(ctx, log.WithField("component", "database"), _env, &noop.Noop{}, aead)
	if err != nil {
		return err
	}

	dbMonitors, err := database.NewMonitors(ctx, _env.DeploymentMode(), dbc)
	if err != nil {
		return err
	}

	dbOpenShiftClusters, err := database.NewOpenShiftClusters(ctx, _env.DeploymentMode(), dbc)
	if err != nil {
		return err
	}

	dbSubscriptions, err := database.NewSubscriptions(ctx, _env.DeploymentMode(), dbc)
	if err != nil {
		return err
	}

	dialer, err := proxy.NewDialer(_env.DeploymentMode())
	if err != nil {
		return err
	}

	mon := pkgmonitor.NewMonitor(log.WithField("component", "monitor"), dialer, dbMonitors, dbOpenShiftClusters, dbSubscriptions, m, clusterm)

	return mon.Run(ctx)
}
