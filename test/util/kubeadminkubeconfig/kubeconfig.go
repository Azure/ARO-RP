package kubeadminkubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

func Get(ctx context.Context, log *logrus.Entry, im instancemetadata.InstanceMetadata, authorizer autorest.Authorizer, resourceID string) (*v1.Config, error) {
	res, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}

	openshiftclusters := redhatopenshift.NewOpenShiftClustersClient(res.SubscriptionID, authorizer)

	oc, err := openshiftclusters.Get(ctx, res.ResourceGroup, res.ResourceName)
	if err != nil {
		return nil, err
	}

	creds, err := openshiftclusters.ListCredentials(ctx, res.ResourceGroup, res.ResourceName)
	if err != nil {
		return nil, err
	}

	tokenURL, err := getTokenURLFromConsoleURL(*oc.OpenShiftClusterProperties.ConsoleProfile.URL)
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

	u, err := url.Parse(*oc.OpenShiftClusterProperties.ApiserverProfile.URL)
	if err != nil {
		return nil, err
	}

	return makeKubeconfig(u.Host, *creds.KubeadminUsername, token, "kube-system"), nil
}

func makeKubeconfig(endpoint, username, token, namespace string) *v1.Config {
	clustername := strings.Replace(endpoint, ".", "-", -1)
	authinfoname := username + "/" + clustername
	contextname := namespace + "/" + clustername + "/" + username

	return &v1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []v1.NamedCluster{
			{
				Name: clustername,
				Cluster: v1.Cluster{
					Server:                "https://" + endpoint,
					InsecureSkipTLSVerify: true,
				},
			},
		},
		AuthInfos: []v1.NamedAuthInfo{
			{
				Name: authinfoname,
				AuthInfo: v1.AuthInfo{
					Token: token,
				},
			},
		},
		Contexts: []v1.NamedContext{
			{
				Name: contextname,
				Context: v1.Context{
					Cluster:   clustername,
					Namespace: namespace,
					AuthInfo:  authinfoname,
				},
			},
		},
		CurrentContext: contextname,
	}
}
