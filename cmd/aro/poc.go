package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Azure/ARO-RP/pkg/poc"
	"github.com/sirupsen/logrus"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func rpPoc(ctx context.Context, log *logrus.Entry, port string) error {
	log.Print("********** ARO-RP on AKS PoC **********")
	var mise = strings.ToLower(enableMISE) == "true"
	ctx, shutdown := context.WithCancel(ctx)
	defer shutdown()
	go handleSigterm(log, shutdown)

	frontEnd := poc.NewFrontend(log, port, mise)

	return frontEnd.Run(ctx)
}

func handleSigterm(log *logrus.Entry, shutdown context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals

	log.Print("received SIGTERM. Terminating...")

	shutdown()
}
