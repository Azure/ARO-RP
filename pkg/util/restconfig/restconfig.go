package restconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/api"
)

// RestConfig returns the Kubernetes *rest.Config for a kubeconfig
func RestConfig(oc *api.OpenShiftCluster) (*rest.Config, error) {
	// must not proceed if PrivateEndpointIP is not set.  In
	// k8s.io/client-go/transport/cache.go, k8s caches our transport, and it
	// can't tell if data in the restconfig.Dial closure has changed.  We don't
	// want it to cache a transport that can never work.
	if oc.Properties.NetworkProfile.APIServerPrivateEndpointIP == "" {
		return nil, errors.New("privateEndpointIP is empty")
	}

	kubeconfig := oc.Properties.AROServiceKubeconfig
	if kubeconfig == nil {
		kubeconfig = oc.Properties.AdminKubeconfig
	}
	config, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return nil, err
	}

	restconfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}

	restconfig.Dial = DialContext(oc)

	return restconfig, nil
}

func DialContext(oc *api.OpenShiftCluster) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" {
			return nil, fmt.Errorf("unimplemented network %q", network)
		}

		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		return (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second}).DialContext(ctx, network, oc.Properties.NetworkProfile.APIServerPrivateEndpointIP+":"+port)
	}
}
