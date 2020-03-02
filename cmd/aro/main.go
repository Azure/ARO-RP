package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

var (
	gitCommit = "unknown"
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "usage: %s {rp,mirror,monitor,deploy}\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) != 1 {
		usage()
		os.Exit(2)
	}

	ctx := context.Background()
	log := utillog.GetLogger()

	log.Printf("starting, git commit %s", gitCommit)

	var err error
	switch strings.ToLower(os.Args[1]) {
	case "mirror":
		err = mirror(ctx, log)
	case "monitor":
		err = monitor(ctx, log)
	case "rp":
		err = rp(ctx, log)
	case "deploy":
		err = deploy(ctx, log)
	default:
		usage()
		os.Exit(2)
	}

	if err != nil {
		log.Fatal(err)
	}
}
