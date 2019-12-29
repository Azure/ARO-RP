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
func RestConfig(ctx context.Context, env env.Interface, doc *api.OpenShiftClusterDocument) (*rest.Config, error) {
	pe, err := env.PrivateEndpoint().Get(ctx, env.ResourceGroup(), "rp-pe-"+doc.ID, "")
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.Load(doc.OpenShiftCluster.Properties.AdminKubeconfig)
	if err != nil {
		return nil, err
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

		return env.DialContext(ctx, network, *(*(*pe.PrivateEndpointProperties.NetworkInterfaces)[0].IPConfigurations)[0].PrivateIPAddress+":"+port)
	}

	return restconfig, nil
}
