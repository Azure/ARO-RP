package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
)

func writeKubeconfig(ctx context.Context, resourceid string) error {
	res, err := azure.ParseResourceID(resourceid)
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
	h := &codec.JsonHandle{
		Indent: 4,
	}

	err = api.AddExtensions(&h.BasicHandle)
	if err != nil {
		return err
	}

	return codec.NewEncoder(os.Stdout, h).Encode(adminKubeconfig)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s resourceid\n", os.Args[0])
		os.Exit(2)
	}
	ctx := context.Background()
	err := writeKubeconfig(ctx, os.Args[1])
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
