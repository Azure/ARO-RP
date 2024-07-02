package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func usage() {
	fmt.Fprint(flag.CommandLine.Output(), "usage:\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  %s deploy config.yaml location\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s gateway\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s mirror [release_image...]\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s monitor\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s portal\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s rp\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s operator {master,worker}\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s update-versions\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s update-role-sets\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	ctx := context.Background()
	audit := utillog.GetAuditEntry()
	log := utillog.GetLogger()

	go func() {
		log.Warn(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Printf("starting, git commit %s", version.GitCommit)

	var err error
	switch strings.ToLower(flag.Arg(0)) {
	case "deploy":
		checkArgs(3)
		err = deploy(ctx, log)
	case "gateway":
		checkArgs(1)
		err = gateway(ctx, log)
	case "mirror":
		checkMinArgs(1)
		err = mirror(ctx, log)
	case "monitor":
		checkArgs(1)
		err = monitor(ctx, log)
	case "rp":
		checkArgs(1)
		err = rp(ctx, log, audit)
	case "portal":
		checkArgs(1)
		err = portal(ctx, log, audit)
	case "operator":
		checkArgs(2)
		err = operator(ctx, log)
	case "update-versions":
		checkArgs(1)
		err = updateOCPVersions(ctx, log)
	case "update-role-sets":
		checkArgs(1)
		err = updatePlatformWorkloadIdentityRoleSets(ctx, log)
	default:
		usage()
		os.Exit(2)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func checkArgs(required int) {
	if len(flag.Args()) != required {
		usage()
		os.Exit(2)
	}
}

func checkMinArgs(required int) {
	if len(flag.Args()) < required {
		usage()
		os.Exit(2)
	}
}

func DBName(isLocalDevelopmentMode bool) (string, error) {
	if !isLocalDevelopmentMode {
		return "ARO", nil
	}

	if err := env.ValidateVars(envDatabaseName); err != nil {
		return "", fmt.Errorf("%v (development mode)", err.Error())
	}

	return os.Getenv(envDatabaseName), nil
}
