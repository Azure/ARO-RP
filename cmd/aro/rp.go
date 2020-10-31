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
	"k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/api/admin"
	_ "github.com/Azure/ARO-RP/pkg/api/v20191231preview"
	_ "github.com/Azure/ARO-RP/pkg/api/v20200430"
	_ "github.com/Azure/ARO-RP/pkg/api/v20201031preview"
	"github.com/Azure/ARO-RP/pkg/backend"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func rp(ctx context.Context, log *logrus.Entry) error {
	_env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	var keys []string
	if _env.DeploymentMode() == deployment.Development {
		keys = []string{
			"PULL_SECRET",
		}
	} else {
		keys = []string{
			"ACR_RESOURCE_ID",
			"ADMIN_API_CLIENT_CERT_COMMON_NAME",
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
		}

		if _, found := os.LookupEnv("PULL_SECRET"); found {
			return fmt.Errorf(`environment variable "PULL_SECRET" set`)
		}
	}
	for _, key := range keys {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	err = _env.InitializeAuthorizers()
	if err != nil {
		return err
	}

	m, err := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"))
	if err != nil {
		return err
	}

	tracing.Register(azure.New(m))
	metrics.Register(metrics.RegisterOpts{
		RequestResult:  k8s.NewResult(m),
		RequestLatency: k8s.NewLatency(m),
	})

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, _env, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(ctx, log.WithField("component", "database"), _env, m, cipher)
	if err != nil {
		return err
	}

	dbAsyncOperations, err := database.NewAsyncOperations(ctx, _env.DeploymentMode(), dbc)
	if err != nil {
		return err
	}

	dbBilling, err := database.NewBilling(ctx, _env.DeploymentMode(), dbc)
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

	go database.EmitMetrics(ctx, log, dbOpenShiftClusters, m)

	feCipher, err := encryption.NewXChaCha20Poly1305(ctx, _env, env.FrontendEncryptionSecretName)
	if err != nil {
		return err
	}

	f, err := frontend.NewFrontend(ctx, log.WithField("component", "frontend"), _env, dbAsyncOperations, dbOpenShiftClusters, dbSubscriptions, api.APIs, m, feCipher, adminactions.New)
	if err != nil {
		return err
	}

	b, err := backend.NewBackend(ctx, log.WithField("component", "backend"), _env, dbAsyncOperations, dbBilling, dbOpenShiftClusters, dbSubscriptions, cipher, m)
	if err != nil {
		return err
	}

	// This part of the code orchestrates shutdown sequence. When sigterm is
	// received, it will trigger backend to stop accepting new documents and
	// finish old ones. Frontend will stop advertising itself to the loadbalancer.
	// When shutdown completes for frontend and backend "/healthz" endpoint
	// will go dark and external observer will know that shutdown sequence is finished
	sigterm := make(chan os.Signal, 1)
	stop := make(chan struct{})
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
