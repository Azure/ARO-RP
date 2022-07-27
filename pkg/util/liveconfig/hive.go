package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func parseKubeconfig(credentials []mgmtcontainerservice.CredentialResult) (*rest.Config, error) {
	res := make([]byte, base64.StdEncoding.DecodedLen(len(*credentials[0].Value)))
	_, err := base64.StdEncoding.Decode(res, *credentials[0].Value)
	if err != nil {
		return nil, err
	}

	clientconfig, err := clientcmd.NewClientConfigFromBytes(res)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func (p *prod) HiveRestConfig(ctx context.Context, index int) (*rest.Config, error) {
	cached, ext := p.cachedCredentials[index]
	if ext {
		return parseKubeconfig(cached)
	}

	rpResourceGroup := fmt.Sprintf("rp-%s", p.location)
	rpResourceName := fmt.Sprintf("aro-aks-cluster-%03d", index)

	res, err := p.managedClusters.ListClusterUserCredentials(ctx, rpResourceGroup, rpResourceName, "")
	if err != nil {
		return nil, err
	}

	p.cachedCredentials[index] = *res.Kubeconfigs
	return parseKubeconfig(*res.Kubeconfigs)
}
