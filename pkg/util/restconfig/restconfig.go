package restconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
)

// RestConfig returns the Kubernetes *rest.Config for a kubeconfig
func RestConfig(env env.Interface, oc *api.OpenShiftCluster) (*rest.Config, error) {
	config, err := clientcmd.Load(oc.Properties.AROServiceKubeconfig)
	if err != nil {
		config, err = clientcmd.Load(oc.Properties.AdminKubeconfig)
		if err != nil {
			return nil, err
		}
	}

	restconfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}

	restconfig.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" {
			return nil, fmt.Errorf("unimplemented network %q", network)
		}

		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		return env.DialContext(ctx, network, oc.Properties.NetworkProfile.PrivateEndpointIP+":"+port)
	}

	return restconfig, nil
}
