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

	"github.com/davecgh/go-spew/spew"
	msgraph_errors "github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	Cluster = "CLUSTER"
)

func run(ctx context.Context, log *logrus.Entry) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: CLUSTER=x %s {create,delete}", os.Args[0])
	}

	if err := env.ValidateVars(Cluster); err != nil {
		return err
	}

	env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	vnetResourceGroup := os.Getenv("RESOURCEGROUP") // TODO: remove this when we deploy and peer a vnet per cluster create
	if os.Getenv("CI") != "" {
		vnetResourceGroup = os.Getenv(Cluster)
	}
	clusterName := os.Getenv(Cluster)

	osClusterVersion := os.Getenv("OS_CLUSTER_VERSION")

	c, err := cluster.New(log, env, os.Getenv("CI") != "")
	if err != nil {
		return err
	}

	switch strings.ToLower(os.Args[1]) {
	case "create":
		return c.Create(ctx, vnetResourceGroup, clusterName, osClusterVersion)
	case "delete":
		return c.Delete(ctx, vnetResourceGroup, clusterName)
	default:
		return fmt.Errorf("invalid command %s", os.Args[1])
	}
}

func main() {
	log := utillog.GetLogger()

	rand.Seed(time.Now().UnixNano())

	if err := run(context.Background(), log); err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			spew.Dump(oDataError.GetError())
		}
		log.Fatal(err)
	}
}
