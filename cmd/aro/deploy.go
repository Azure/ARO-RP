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

	pkgdeploy "github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func deploy(ctx context.Context, log *logrus.Entry) error {
	for _, key := range []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	env, err := env.NewCoreForCI(ctx, log)
	if err != nil {
		return err
	}

	deployVersion, location := version.GitCommit, flag.Arg(2)

	log.Printf("deploying version %s to location %s", deployVersion, location)

	if deployVersion == "unknown" ||
		(!env.IsDevelopmentMode() && strings.Contains(deployVersion, "dirty")) {
		return fmt.Errorf("invalid deploy version %q", deployVersion)
	}

	if strings.ToLower(location) != location {
		return fmt.Errorf("location %s must be lower case", location)
	}

	config, err := pkgdeploy.GetConfig(flag.Arg(1), location)
	if err != nil {
		return err
	}

	deployer, err := pkgdeploy.New(ctx, log, env, config, deployVersion)
	if err != nil {
		return err
	}

	err = deployer.PreDeploy(ctx)
	if err != nil {
		return err
	}

	err = deployer.DeployRP(ctx)
	if err != nil {
		return err
	}

	err = deployer.UpgradeRP(ctx)
	if err != nil {
		return err
	}

	// Must be last step so we can be sure there are no RPs at older versions
	// still serving
	return deployer.SaveVersion(ctx)
}
