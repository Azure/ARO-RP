package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/cluster"
	msgraph_errors "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	Cluster = "CLUSTER"
)

func run(ctx context.Context, log *logrus.Entry) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: CLUSTER=x %s {create,delete}", os.Args[0])
	}

	conf, err := cluster.NewClusterConfigFromEnv()
	if err != nil {
		return err
	}

	c, err := cluster.New(log, conf)
	if err != nil {
		return err
	}

	spew.Dump(c.Config)
	switch strings.ToLower(os.Args[1]) {
	case "create":
		return c.Create(ctx)
	case "delete":
		return c.Delete(ctx, conf.VnetResourceGroup, conf.ClusterName)
	default:
		return fmt.Errorf("invalid command %s", os.Args[1])
	}
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			spew.Dump(oDataError.GetErrorEscaped())
		}
		log.Fatal(err)
	}
}
