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
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/api/admin"
	_ "github.com/Azure/ARO-RP/pkg/api/v20191231preview"
	_ "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/backend"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func rp(ctx context.Context, log *logrus.Entry) error {
	uuid := uuid.NewV4().String()
	log.Printf("uuid %s", uuid)

	_env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	var keys []string
	if _, ok := _env.(env.Dev); ok {
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
	metrics.Register(k8s.NewLatency(m), k8s.NewResult(m))

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, _env)
	if err != nil {
		return err
	}

	db, err := database.NewDatabase(ctx, log.WithField("component", "database"), _env, m, cipher, uuid)
	if err != nil {
		return err
	}

	f, err := frontend.NewFrontend(ctx, log.WithField("component", "frontend"), _env, db, api.APIs, m)
	if err != nil {
		return err
	}

	b, err := backend.NewBackend(ctx, log.WithField("component", "backend"), _env, db, m)
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
