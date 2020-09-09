package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/test/util/kubeconfig"
)

func run(ctx context.Context, log *logrus.Entry, resourceID string) error {
	res, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	return kubeconfig.NewManager(log, res.SubscriptionID, authorizer).Print(ctx, res.ResourceGroup, res.ResourceName)
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
