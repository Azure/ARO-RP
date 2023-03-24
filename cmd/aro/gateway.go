package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	pkgdbtoken "github.com/Azure/ARO-RP/pkg/dbtoken"
	"github.com/Azure/ARO-RP/pkg/env"
	pkggateway "github.com/Azure/ARO-RP/pkg/gateway"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
)

func gateway(ctx context.Context, log *logrus.Entry) error {
	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	for _, key := range []string{
		"AZURE_DBTOKEN_CLIENT_ID",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	m := statsd.New(ctx, log.WithField("component", "gateway"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(log.WithField("component", "gateway"), m)
	if err != nil {
		return err
	}

	go g.Run()

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, nil, m, nil)
	if err != nil {
		return err
	}

	resource := os.Getenv("AZURE_DBTOKEN_CLIENT_ID")
	msiRefresherAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextGateway, resource)
	if err != nil {
		return err
	}

	// TODO: refactor this poor man's feature flag
	insecureSkipVerify := _env.IsLocalDevelopmentMode()
	for _, feature := range strings.Split(os.Getenv("GATEWAY_FEATURES"), ",") {
		if feature == "InsecureSkipVerifyDBTokenCertificate" {
			insecureSkipVerify = true
			break
		}
	}

	dbRefresher, err := pkgdbtoken.NewRefresher(log, _env, msiRefresherAuthorizer, insecureSkipVerify, dbc, "gateway", m, "gateway")
	if err != nil {
		return err
	}

	dbGateway, err := database.NewGateway(ctx, _env.IsLocalDevelopmentMode(), dbc)
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
