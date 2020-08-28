package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"

	"github.com/Azure/ARO-RP/pkg/proxy"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

var (
	certFile       = flag.String("certFile", "secrets/proxy.crt", "file containing server certificate")
	keyFile        = flag.String("keyFile", "secrets/proxy.key", "file containing server key")
	clientCertFile = flag.String("clientCertFile", "secrets/proxy-client.crt", "file containing client certificate")
	subnet         = flag.String("subnet", "10.0.0.0/8", "allowed subnet")
)

func run() error {
	s := &proxy.Server{
		CertFile:       *certFile,
		KeyFile:        *keyFile,
		ClientCertFile: *clientCertFile,
		Subnet:         *subnet,
	}

	return s.Run()
}

func main() {
	log := utillog.GetLogger()

	log.Printf("starting, git commit %s", version.GitCommit)

	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}
