package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	deployer "github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func deploy(ctx context.Context, log *logrus.Entry) error {
	deployVersion, location := version.GitCommit, flag.Arg(2)

	if os.Getenv("RP_VERSION") != "" {
		deployVersion = os.Getenv("RP_VERSION")
	}

	log.Printf("deploying version %s to location %s", deployVersion, location)

	if deployVersion == "unknown" || strings.Contains(deployVersion, "dirty") {
		return fmt.Errorf("invalid deploy version %q", deployVersion)
	}

	if strings.ToLower(location) != location {
		return fmt.Errorf("location %s must be lower case", location)
	}

	config, err := deployer.GetConfig(flag.Arg(1), location)
	if err != nil {
		return err
	}

	deployer, err := deployer.New(ctx, log, config, deployVersion)
	if err != nil {
		return err
	}

	rpServicePrincipalID, err := deployer.PreDeploy(ctx)
	if err != nil {
		return err
	}

	// if pre-deploy is set, we terminate early. This is so we could populate
	// vault and other configurations, required for main deployment. This is usually
	// day 1 deployment setting only
	if os.Getenv("RP_PREDEPLOY_ONLY") != "" {
		return nil
	}

	err = deployer.Deploy(ctx, rpServicePrincipalID)
	if err != nil {
		return err
	}

	return deployer.Upgrade(ctx)
}
