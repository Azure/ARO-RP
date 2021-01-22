package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func run(ctx context.Context, log *logrus.Entry) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: CLUSTER=x %s {create,delete}", os.Args[0])
	}

	for _, key := range []string{
		"CLUSTER",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	// assume they're the same unless passed in explicitly
	_, found := os.LookupEnv("CLUSTER_RESOURCEGROUP")
	if !found {
		os.Setenv("CLUSTER_RESOURCEGROUP", os.Getenv("CLUSTER"))
	}

	localEnv, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	clusterEnv := env.DeriveCoreForCluster(localEnv)

	c, err := cluster.New(log, localEnv, clusterEnv, os.Getenv("CI") != "")
	if err != nil {
		return err
	}

	switch strings.ToLower(os.Args[1]) {
	case "create":
		return c.Create(ctx, os.Getenv("CLUSTER"))
	case "delete":
		return c.Delete(ctx, os.Getenv("CLUSTER"))
	default:
		return fmt.Errorf("invalid command %s", os.Args[1])
	}
}

func main() {
	log := utillog.GetLogger()

	rand.Seed(time.Now().UnixNano())

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
