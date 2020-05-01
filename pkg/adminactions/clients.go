package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	samplesclient "github.com/openshift/cluster-samples-operator/pkg/generated/clientset/versioned"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// InitializeClients initializes clients.
func (a *adminactions) InitializeClients(ctx context.Context) error {
	restConfig, err := restconfig.RestConfig(a.env, a.oc)
	if err != nil {
		return err
	}

	a.discoverycli, err = discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return err
	}

	a.dynamiccli, err = dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	a.cli, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	a.configcli, err = configclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	a.operatorcli, err = operatorclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	a.samplescli, err = samplesclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	a.seccli, err = securityclient.NewForConfig(restConfig)
	return err
}
