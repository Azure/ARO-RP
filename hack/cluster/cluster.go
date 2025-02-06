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

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	msgraph_errors "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
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

	env, err := env.NewCore(ctx, log, env.COMPONENT_TOOLING)
	if err != nil {
		return err
	}

	vnetResourceGroup := os.Getenv("RESOURCEGROUP") // TODO: remove this when we deploy and peer a vnet per cluster create
	if os.Getenv("CI") != "" {
		vnetResourceGroup = os.Getenv(Cluster)
	}
	clusterName := os.Getenv(Cluster)

	osClusterVersion := os.Getenv("OS_CLUSTER_VERSION")
	if osClusterVersion == "" {
		osClusterVersion = version.DefaultInstallStream.Version.String()
		log.Infof("using default cluster version %s", osClusterVersion)
	} else {
		log.Infof("using specified cluster version %s", osClusterVersion)
	}

	masterVmSize := os.Getenv("MASTER_VM_SIZE")
	if masterVmSize == "" {
		masterVmSize = cluster.DefaultMasterVmSize.String()
		log.Infof("using default master VM size %s", masterVmSize)
	} else {
		log.Infof("using specified master VM size %s", masterVmSize)
	}

	workerVmSize := os.Getenv("WORKER_VM_SIZE")
	if workerVmSize == "" {
		workerVmSize = cluster.DefaultWorkerVmSize.String()
		log.Infof("using default worker VM size %s", workerVmSize)
	} else {
		log.Infof("using specified worker VM size %s", workerVmSize)
	}

	c, err := cluster.New(log, env, os.Getenv("CI") != "")
	if err != nil {
		return err
	}

	switch strings.ToLower(os.Args[1]) {
	case "create":
		return c.Create(ctx, vnetResourceGroup, clusterName, osClusterVersion, masterVmSize, workerVmSize)
	case "delete":
		return c.Delete(ctx, vnetResourceGroup, clusterName)
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
