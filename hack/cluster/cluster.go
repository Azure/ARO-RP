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
	"github.com/spf13/viper"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	msgraph_errors "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	Cluster = "CLUSTER"
)

func run(ctx context.Context, log *logrus.Entry, cfg *viper.Viper) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: CLUSTER=x %s {create,createApp,deleteApp,delete}", os.Args[0])
	}

	_env, err := env.NewCore(ctx, log, env.COMPONENT_TOOLING, cfg)
	if err != nil {
		return err
	}

	if err := _env.ValidateVars(Cluster); err != nil {
		return err
	}

	vnetResourceGroup := _env.GetEnv("RESOURCEGROUP") // TODO: remove this when we deploy and peer a vnet per cluster create
	if _env.IsCI() {
		vnetResourceGroup = _env.GetEnv(Cluster)
	}
	clusterName := _env.GetEnv(Cluster)

	osClusterVersion := _env.GetEnv("OS_CLUSTER_VERSION")
	if osClusterVersion == "" {
		osClusterVersion = version.DefaultInstallStream.Version.String()
		log.Infof("using default cluster version %s", osClusterVersion)
	} else {
		log.Infof("using specified cluster version %s", osClusterVersion)
	}

	c, err := cluster.New(log, _env)
	if err != nil {
		return err
	}

	switch strings.ToLower(os.Args[1]) {
	case "create":
		return c.Create(ctx, vnetResourceGroup, clusterName, osClusterVersion)
	case "createapp":
		return c.CreateApp(ctx, clusterName)
	case "deleteapp":
		return c.DeleteApp(ctx)
	case "delete":
		return c.Delete(ctx, vnetResourceGroup, clusterName)
	default:
		return fmt.Errorf("invalid command %s", os.Args[1])
	}
}

func main() {
	log := utillog.GetLogger()
	cfg := viper.GetViper()
	cfg.AutomaticEnv()

	if err := run(context.Background(), log, cfg); err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			spew.Dump(oDataError.GetErrorEscaped())
		}
		log.Fatal(err)
	}
}
