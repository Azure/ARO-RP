package kubeadminkubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/env"
	redhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2024-08-12-preview/redhatopenshift"
)

func Get(ctx context.Context, log *logrus.Entry, env env.Core, authorizer autorest.Authorizer, resourceID string) (*clientcmdv1.Config, error) {
	res, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}

	openshiftclusters := redhatopenshift20240812preview.NewOpenShiftClustersClient(env.Environment(), res.SubscriptionID, authorizer)

	oc, err := openshiftclusters.Get(ctx, res.ResourceGroup, res.ResourceName)
	if err != nil {
		return nil, err
	}

	creds, err := openshiftclusters.ListCredentials(ctx, res.ResourceGroup, res.ResourceName)
	if err != nil {
		return nil, err
	}

	tokenURL, err := getTokenURLFromConsoleURL(*oc.ConsoleProfile.URL)
	if err != nil {
		return nil, err
	}

	var token string

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
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
		return nil, err
	}

	u, err := url.Parse(*oc.ApiserverProfile.URL)
	if err != nil {
		return nil, err
	}

	return makeKubeconfig(u.Host, *creds.KubeadminUsername, token, "kube-system"), nil
}

func makeKubeconfig(endpoint, username, token, namespace string) *clientcmdv1.Config {
	clustername := strings.ReplaceAll(endpoint, ".", "-")
	authinfoname := username + "/" + clustername
	contextname := namespace + "/" + clustername + "/" + username

	return &clientcmdv1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: clustername,
				Cluster: clientcmdv1.Cluster{
					Server:                "https://" + endpoint,
					InsecureSkipTLSVerify: true,
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{
				Name: authinfoname,
				AuthInfo: clientcmdv1.AuthInfo{
					Token: token,
				},
			},
		},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: contextname,
				Context: clientcmdv1.Context{
					Cluster:   clustername,
					Namespace: namespace,
					AuthInfo:  authinfoname,
				},
			},
		},
		CurrentContext: contextname,
	}
}
