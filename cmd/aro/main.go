package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func usage() {
	fmt.Fprint(flag.CommandLine.Output(), "usage:\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  %s dbtoken\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s deploy config.yaml location\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s gateway\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s mirror\n", os.Args[0])

	// indent the mirror flags... probably a better way to do this? :)
	s := &strings.Builder{}
	mirror := mirrorFlags()
	mirror.SetOutput(s)
	mirror.PrintDefaults()
	for _, l := range strings.Split(s.String(), "\n") {
		if l == "" {
			continue
		}
		fmt.Fprintf(flag.CommandLine.Output(), "  %s\n", l)
	}

	fmt.Fprintf(flag.CommandLine.Output(), "  %s monitor\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s portal\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s rp\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s operator {master,worker}\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s update-versions\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Usage = usage
	flag.Parse()

	ctx := context.Background()
	audit := utillog.GetAuditEntry()
	log := utillog.GetLogger()

	// TODO: Use `azuretypes.Platform.IsARO()` from github.com/openshift/installer/pkg/types/azure
	if !platformIsAro {
		log.Fatal("ARO-RP must be built, run, and tested with '-tags aro' to support github.com/openshift/installer, see https://github.com/openshift/installer/pull/4843/files")
	}

	go func() {
		log.Warn(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Printf("starting, git commit %s", version.GitCommit)

	var err error
	switch strings.ToLower(flag.Arg(0)) {
	case "dbtoken":
		checkArgs(1)
		err = dbtoken(ctx, log)
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
