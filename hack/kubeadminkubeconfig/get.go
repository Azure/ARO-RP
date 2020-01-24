package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func writeKubeconfig(ctx context.Context, resourceID string) error {
	res, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	openshiftclusters := redhatopenshift.NewOpenShiftClustersClient(res.SubscriptionID, authorizer)

	oc, err := openshiftclusters.Get(ctx, res.ResourceGroup, res.ResourceName)
	if err != nil {
		return err
	}

	creds, err := openshiftclusters.ListCredentials(ctx, res.ResourceGroup, res.ResourceName)
	if err != nil {
		return err
	}

	tokenURL, err := getTokenURLFromConsoleURL(*oc.Properties.ConsoleProfile.URL)
	if err != nil {
		return err
	}

	token, err := getAuthorizedToken(tokenURL, *creds.KubeadminUsername, *creds.KubeadminPassword)
	if err != nil {
		return err
	}

	adminKubeconfig, err := makeKubeconfig(strings.Replace(*oc.Properties.ApiserverProfile.URL, "https://", "", 1), *creds.KubeadminUsername, token, "kube-system")
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

	err := writeKubeconfig(ctx, os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
}
