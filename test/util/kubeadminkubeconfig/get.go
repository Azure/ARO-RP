package kubeadminkubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"log"
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift"
)

func GetKubeconfig(ctx context.Context, aro redhatopenshift.OpenShiftCluster, cred redhatopenshift.OpenShiftClusterCredentials) (*v1.Config, error) {

	tokenURL, err := getTokenURLFromConsoleURL(*aro.OpenShiftClusterProperties.ConsoleProfile.URL)
	if err != nil {
		return nil, err
	}

	var token string

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	err = wait.PollImmediateUntil(time.Second, func() (bool, error) {
		token, err = getAuthorizedToken(ctx, tokenURL, *cred.KubeadminUsername, *cred.KubeadminPassword)
		if err != nil {
			log.Print(err)
			return false, nil
		}

		return true, nil
	}, timeoutCtx.Done())
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(*aro.OpenShiftClusterProperties.ApiserverProfile.URL)
	if err != nil {
		return nil, err
	}

	adminKubeconfig := makeKubeconfig(u.Host, *cred.KubeadminUsername, token, "kube-system")

	return adminKubeconfig, nil
}
