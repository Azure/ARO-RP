package restconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"
	"time"

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
		return (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext(ctx, network, *(*(*pe.PrivateEndpointProperties.NetworkInterfaces)[0].IPConfigurations)[0].PrivateIPAddress)
	}

	return restconfig, nil
}
