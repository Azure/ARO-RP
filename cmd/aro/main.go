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
)

var (
	gitCommit = "unknown"
)

func usage() {
	fmt.Fprint(flag.CommandLine.Output(), "usage: \n")
	fmt.Fprintf(flag.CommandLine.Output(), "       %s rp \n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "       %s mirror\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "       %s monitor\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "       %s deploy {name} {config_file_path}\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Usage = usage
	flag.Parse()

	ctx := context.Background()
	log := utillog.GetLogger()

	log.Printf("starting, git commit %s", gitCommit)

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
