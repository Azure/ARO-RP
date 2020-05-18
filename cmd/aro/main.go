package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func usage() {
	fmt.Fprint(flag.CommandLine.Output(), "usage:\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  %s deploy config.yaml location\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s mirror\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s monitor\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s rp\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  %s operator\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Usage = usage
	flag.Parse()

	ctx := context.Background()
	log := utillog.GetLogger()

	log.Printf("starting, git commit %s", version.GitCommit)

	var err error
	switch strings.ToLower(flag.Arg(0)) {
	case "mirror":
		checkArgs(1)
		err = mirror(ctx, log)
	case "monitor":
		checkArgs(1)
		err = monitor(ctx, log)
	case "rp":
		checkArgs(1)
		err = rp(ctx, log)
	case "deploy":
		checkArgs(3)
		err = deploy(ctx, log)
	case "operator":
		checkArgs(1)
		err = operator(ctx, log)
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
