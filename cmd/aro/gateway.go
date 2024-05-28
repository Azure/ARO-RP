package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	pkggateway "github.com/Azure/ARO-RP/pkg/gateway"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
)

func gateway(ctx context.Context, log *logrus.Entry) error {
	_env, err := env.NewCore(ctx, log, env.COMPONENT_GATEWAY)
	if err != nil {
		return err
	}

	if err = env.ValidateVars("AZURE_DBTOKEN_CLIENT_ID"); err != nil {
		return err
	}

	m := statsd.New(ctx, log.WithField("component", "gateway"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(log.WithField("component", "gateway"), m)
	if err != nil {
		return err
	}

	go g.Run()

	if err := env.ValidateVars(envDatabaseAccountName); err != nil {
		return err
	}

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return err
	}
	logrusEntry := log.WithField("component", "database")

	dbName, err := DBName(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	dbAccountName := os.Getenv(envDatabaseAccountName)
	dbAuthorizer, err := database.NewTokenAuthorizer(ctx, logrusEntry, msiToken, dbAccountName, []string{})
	if err != nil {
		return err
	}
	dbc, err := database.NewDatabaseClient(logrusEntry, _env, dbAuthorizer, m, nil, dbAccountName)
	if err != nil {
		return err
	}

	dbGateway, err := database.NewGateway(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	httpl, err := utilnet.Listen("tcp", ":8080", pkggateway.SocketSize)
	if err != nil {
		return err
	}

	httpsl, err := utilnet.Listen("tcp", ":8443", pkggateway.SocketSize)
	if err != nil {
		return err
	}

	healthListener, err := utilnet.Listen("tcp", ":8081", pkggateway.SocketSize)
	if err != nil {
		return err
	}

	log.Print("listening")

	p, err := pkggateway.NewGateway(ctx, _env, log.WithField("component", "gateway"), log.WithField("component", "gateway-access"), dbGateway, httpsl, httpl, healthListener, os.Getenv("ACR_RESOURCE_ID"), os.Getenv("GATEWAY_DOMAINS"), m)
	if err != nil {
		return err
	}

	sigterm := make(chan os.Signal, 1)
	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	signal.Notify(sigterm, syscall.SIGTERM)

	go p.Run(cancelCtx, done)

	<-sigterm
	log.Print("received SIGTERM")
	cancel()
	<-done

	return nil
}
