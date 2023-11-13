package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/spf13/pflag"
)

var (
	serverPort string
)

func init() {
	pflag.StringVar(&serverPort, "server-port", "8080", "port to service http requests")
}

func main() {
	pflag.Parse()

	log := utillog.GetLogger()

	ctx := context.Background()
	if err := rpPoc(ctx, log, serverPort); err != nil {
		log.Fatal(err)
	}
}

func DBName(isLocalDevelopmentMode bool) (string, error) {
	if !isLocalDevelopmentMode {
		return "ARO", nil
	}

	if err := env.ValidateVars(DatabaseName); err != nil {
		return "", fmt.Errorf("%v (development mode)", err.Error())
	}

	return os.Getenv(DatabaseName), nil
}
