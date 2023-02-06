package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func run(ctx context.Context, log *logrus.Entry) error {
	vars := []string{
		"ADMIN_OBJECT_ID",
		"AZURE_CLIENT_ID",
		"AZURE_DBTOKEN_CLIENT_ID",
		"AZURE_SERVICE_PRINCIPAL_ID",
		"AZURE_FP_SERVICE_PRINCIPAL_ID",
		"AZURE_PORTAL_ACCESS_GROUP_IDS",
		"AZURE_PORTAL_CLIENT_ID",
		"AZURE_PORTAL_ELEVATED_GROUP_IDS",
		"HOME",
		"PARENT_DOMAIN_NAME",
		"USER",
	}
	if err := validateEnvVars(vars...); err != nil {
		return err
	}

	if _, found := os.LookupEnv("SSH_PUBLIC_KEY"); !found {
		log.Warnf("environment variable SSH_PUBLIC_KEY unset, will use %s/.ssh/id_rsa.pub", os.Getenv("HOME"))
	}

	env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	c, err := deploy.DevConfig(env)
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(b)
	return err
}

// validateEnvVars iterates over all the elements of vars and
// if it does not exist an environment variable with that name, it will return an error.
// Otherwise it returns nil.
func validateEnvVars(vars ...string) error {
	for _, v := range vars {
		if _, found := os.LookupEnv(v); !found {
			return fmt.Errorf("environment variable %q unset", v)
		}
	}
	return nil
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
