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

	"github.com/Azure/ARO-RP/pkg/util/cluster"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
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

	deploymentMode := deployment.NewMode()
	log.Infof("running in %s mode", deploymentMode)

	instancemetadata, err := instancemetadata.NewDev()
	if err != nil {
		return nil
	}

	c, err := cluster.New(log, deploymentMode, instancemetadata, false)
	if err != nil {
		return nil
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
