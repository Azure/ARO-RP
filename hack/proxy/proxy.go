package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"os"

	"github.com/Azure/ARO-RP/pkg/proxy"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

var (
	certFile       = flag.String("certFile", "proxy.crt", "file containing server certificate")
	keyFile        = flag.String("keyFile", "proxy.key", "file containing server key")
	clientCertFile = flag.String("clientCertFile", "proxy-client.crt", "file containing client certificate")
	subnet         = flag.String("subnet", "10.0.0.0/8", "allowed subnet")
)

func main() {
	log := utillog.GetLogger()

	log.Printf("starting, git commit %s", version.GitCommit)

	flag.Parse()

	secretsDir := os.Getenv("SECRETS")

	s := &proxy.Server{
		CertFile:       secretsDir + "/" + *certFile,
		KeyFile:        secretsDir + "/" + *keyFile,
		ClientCertFile: secretsDir + "/" + *clientCertFile,
		Subnet:         *subnet,
	}

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
