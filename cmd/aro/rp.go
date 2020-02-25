package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/api/admin"
	_ "github.com/Azure/ARO-RP/pkg/api/v20191231preview"
	"github.com/Azure/ARO-RP/pkg/backend"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func rp(ctx context.Context, log *logrus.Entry) error {
	for _, key := range []string{
		"PULL_SECRET",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	uuid := uuid.NewV4().String()
	log.Printf("uuid %s", uuid)

	env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	m, err := statsd.New(ctx, log.WithField("component", "metrics"), env)
	if err != nil {
		return err
	}

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, env)
	if err != nil {
		return err
	}

	db, err := database.NewDatabase(ctx, log.WithField("component", "database"), env, m, cipher, uuid)
	if err != nil {
		return err
	}

	sigterm := make(chan os.Signal, 1)
	stop := make(chan struct{})
	doneB := make(chan struct{})
	doneF := make(chan struct{})
	signal.Notify(sigterm, syscall.SIGTERM)

	f, err := frontend.NewFrontend(ctx, log.WithField("component", "frontend"), env, db, api.APIs, m)
	if err != nil {
		return err
	}

	b, err := backend.NewBackend(ctx, log.WithField("component", "backend"), env, db, m)
	if err != nil {
		return err
	}

	log.Print("listening")

	go b.Run(ctx, stop, doneB)
	go f.Run(ctx, stop, doneF)

	<-sigterm
	log.Print("received SIGTERM")
	close(stop)
	<-doneF
	<-doneB

	return nil
}
