package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Azure/ARO-RP/pkg/database"
	pkgdbtoken "github.com/Azure/ARO-RP/pkg/dbtoken"
	"github.com/Azure/ARO-RP/pkg/env"
	pkggateway "github.com/Azure/ARO-RP/pkg/gateway"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
)

func gateway(ctx context.Context, log *logrus.Entry, cfg *viper.Viper) error {
	_env, err := env.NewCore(ctx, log, env.COMPONENT_GATEWAY, cfg)
	if err != nil {
		return err
	}

	if err = _env.ValidateVars("AZURE_DBTOKEN_CLIENT_ID"); err != nil {
		return err
	}

	m := statsd.New(ctx, log.WithField("component", "gateway"), _env, _env.GetEnv("MDM_ACCOUNT"), _env.GetEnv("MDM_NAMESPACE"), _env.GetEnv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(log.WithField("component", "gateway"), m)
	if err != nil {
		return err
	}

	go g.Run()

	if err := _env.ValidateVars(envDatabaseAccountName); err != nil {
		return err
	}
	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, nil, m, nil, _env.GetEnv(envDatabaseAccountName))
	if err != nil {
		return err
	}

	// Access token GET request needs to be:
	// http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=$AZURE_DBTOKEN_CLIENT_ID
	//
	// In this context, the "resource" parameter is passed to azidentity as a
	// "scope" argument even though a scope normally consists of an endpoint URL.
	scope := _env.GetEnv("AZURE_DBTOKEN_CLIENT_ID")
	msiRefresherAuthorizer, err := _env.NewMSIAuthorizer(scope)
	if err != nil {
		return err
	}

	// TODO: refactor this poor man's feature flag
	insecureSkipVerify := _env.IsLocalDevelopmentMode()
	for _, feature := range strings.Split(_env.GetEnv("GATEWAY_FEATURES"), ",") {
		if feature == "InsecureSkipVerifyDBTokenCertificate" {
			insecureSkipVerify = true
			break
		}
	}

	url, err := getURL(_env)
	if err != nil {
		return err
	}
	dbRefresher := pkgdbtoken.NewRefresher(log, _env, msiRefresherAuthorizer, insecureSkipVerify, dbc, "gateway", m, "gateway", url)

	dbName, err := DBName(_env)
	if err != nil {
		return err
	}

	dbGateway, err := database.NewGateway(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	go func() {
		_ = dbRefresher.Run(ctx)
	}()

	log.Print("waiting for database token")
	for !dbRefresher.HasSyncedOnce() {
		time.Sleep(time.Second)
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

	p, err := pkggateway.NewGateway(ctx, _env, log.WithField("component", "gateway"), log.WithField("component", "gateway-access"), dbGateway, httpsl, httpl, healthListener, _env.GetEnv("ACR_RESOURCE_ID"), _env.GetEnv("GATEWAY_DOMAINS"), m)
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

func getURL(_env env.Core) (string, error) {
	if _env.IsLocalDevelopmentMode() {
		return "https://localhost:8445", nil
	}

	if err := _env.ValidateVars(envDBTokenUrl); err != nil {
		return "", err
	}

	return _env.GetEnv(envDBTokenUrl), nil
}
