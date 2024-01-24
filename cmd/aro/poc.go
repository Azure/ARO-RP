package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Azure/ARO-RP/pkg/poc"
	"github.com/sirupsen/logrus"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func rpPoc(ctx context.Context, log *logrus.Entry) error {
	log.Print("********** ARO-RP on AKS PoC **********")
	ctx, shutdown := context.WithCancel(ctx)
	defer shutdown()
	go handleSigterm(log, shutdown)

	config := poc.FrontendConfig{
		Port:       serverPort,
		EnableMISE: enableMISE,
	}

	frontEnd := poc.NewFrontend(log, config)

	return frontEnd.Run(ctx)
}

func handleSigterm(log *logrus.Entry, shutdown context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals

	log.Print("received SIGTERM. Terminating...")

	shutdown()
}
