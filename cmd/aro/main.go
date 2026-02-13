package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	_ "net/http/pprof"

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
	fmt.Fprintf(flag.CommandLine.Output(), "  %s mimo-actuator\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s mimo-scheduler\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	ctx := context.Background()
	serviceName := serviceForCommand(flag.Arg(0))
	log := env.LoggerForService(serviceName, utillog.GetLogger())

	go func() {
		log.Warn(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Printf("starting, git commit %s", version.GitCommit)
	log.Printf("command line: '%s'", strings.Join(os.Args, " "))

	var err error
	switch serviceName {
	case env.SERVICE_DEPLOY:
		checkArgs(3)
		err = deploy(ctx, log)
	case env.SERVICE_GATEWAY:
		checkArgs(1)
		err = gateway(ctx, log)
	case env.SERVICE_MIRROR:
		checkMinArgs(1)
		err = mirror(ctx, log)
	case env.SERVICE_MONITOR:
		checkArgs(1)
		err = monitor(ctx, log)
	case env.SERVICE_RP:
		checkArgs(1)
		audit := utillog.GetAuditEntry()
		err = rp(ctx, log, audit)
	case env.SERVICE_PORTAL:
		checkArgs(1)
		audit := utillog.GetAuditEntry()
		err = portal(ctx, log, audit)
	case env.SERVICE_OPERATOR:
		checkArgs(2)
		err = operator(ctx, log)
	case env.SERVICE_UPDATE_OCP_VERSIONS:
		checkArgs(1)
		err = updateOCPVersions(ctx, log)
	case env.SERVICE_UPDATE_ROLE_SETS:
		checkArgs(1)
		err = updatePlatformWorkloadIdentityRoleSets(ctx, log)
	case env.SERVICE_MIMO_ACTUATOR:
		checkArgs(1)
		err = mimoActuator(ctx, log)
	case env.SERVICE_MIMO_SCHEDULER:
		checkArgs(1)
		err = mimoScheduler(ctx, log)
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

func serviceForCommand(cmd string) env.ServiceName {
	switch strings.ToLower(cmd) {
	case "deploy":
		return env.SERVICE_DEPLOY
	case "gateway":
		return env.SERVICE_GATEWAY
	case "mirror":
		return env.SERVICE_MIRROR
	case "monitor":
		return env.SERVICE_MONITOR
	case "rp":
		return env.SERVICE_RP
	case "portal":
		return env.SERVICE_PORTAL
	case "operator":
		return env.SERVICE_OPERATOR
	case "update-versions":
		return env.SERVICE_UPDATE_OCP_VERSIONS
	case "update-role-sets":
		return env.SERVICE_UPDATE_ROLE_SETS
	case "mimo-actuator":
		return env.SERVICE_MIMO_ACTUATOR
	case "mimo-scheduler":
		return env.SERVICE_MIMO_SCHEDULER
	}
	return ""
}
