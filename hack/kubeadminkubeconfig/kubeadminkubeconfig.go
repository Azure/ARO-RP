package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"os"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/test/util/kubeadminkubeconfig"
)

func run(ctx context.Context, log *logrus.Entry, resourceID string) error {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	adminKubeconfig, err := kubeadminkubeconfig.Get(ctx, log, authorizer, resourceID)
	if err != nil {
		return err
	}

	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "    ")
	return e.Encode(adminKubeconfig)
}

func main() {
	ctx := context.Background()
	log := utillog.GetLogger()

	if len(os.Args) != 2 {
		log.Fatalf("usage: %s resourceid\n", os.Args[0])
	}

	err := run(ctx, log, os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
}
