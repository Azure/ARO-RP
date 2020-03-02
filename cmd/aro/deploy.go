package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	deployer "github.com/Azure/ARO-RP/pkg/deploy"
)

func deploy(ctx context.Context, log *logrus.Entry) error {
	for _, key := range []string{
		"LOCATION",
		"AZURE_TENANT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_CLIENT_ID",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_RP_PARAMETERS_FILE",
		"AZURE_RP_RESOURCEGROUP_NAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	if len(os.Getenv("AZURE_RP_VERSION")) > 0 {
		gitCommit = os.Getenv("AZURE_RP_VERSION")
	} else if gitCommit == "unknown" ||
		strings.Contains(gitCommit, "dirty") {
		return fmt.Errorf("gitCommit '%s' is not valid deployment version", gitCommit)
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	deployer, err := deployer.New(ctx, log, authorizer, gitCommit)
	if err != nil {
		return err
	}

	rpServicePrincipalID, err := deployer.PreDeploy(ctx, log)
	if err != nil {
		return err
	}
	// if pre-deploy is set, we terminate early. This is so we could populate
	// vault and other configurations, required for main deployment. This is usually
	// day 1 deployment setting only
	if len(os.Getenv("AZURE_RP_PREDEPLOY_ONLY")) > 0 {
		return nil
	}

	err = deployer.Deploy(ctx, log, rpServicePrincipalID)
	if err != nil {
		return err
	}

	err = deployer.Upgrade(ctx, log)
	if err != nil {
		return err
	}

	return nil
}
