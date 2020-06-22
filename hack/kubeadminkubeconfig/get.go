package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func writeKubeconfig(ctx context.Context, log *logrus.Entry, resourceID string) error {
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

	tokenURL, err := getTokenURLFromConsoleURL(*oc.OpenShiftClusterProperties.ConsoleProfile.URL)
	if err != nil {
		return err
	}

	var token string

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	err = wait.PollImmediateUntil(time.Second, func() (bool, error) {
		token, err = getAuthorizedToken(ctx, tokenURL, *creds.KubeadminUsername, *creds.KubeadminPassword)
		if err != nil {
			log.Print(err)
			return false, nil
		}

		return true, nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	u, err := url.Parse(*oc.OpenShiftClusterProperties.ApiserverProfile.URL)
	if err != nil {
		return err
	}

	adminKubeconfig := makeKubeconfig(u.Host, *creds.KubeadminUsername, token, "kube-system")

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

	err := writeKubeconfig(ctx, log, os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
}
