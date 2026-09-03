package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/api"
)

// InstallerRestConfig returns the Kubernetes RestConfig for the installer execution cluster
// based on the selected backend and environment.
//
// For backend AKSJob:
//   - Production/INT: Returns RestConfig for the regional SVC AKS cluster
//   - CI: Returns RestConfig for the existing Hive-host AKS cluster (retained for CI)
//   - Dev: Can use either based on configuration
//
// For backend Hive:
//   - Returns the Hive AKS cluster RestConfig (same as HiveRestConfig)
//
// For backend Podman:
//   - Returns an error (Podman does not use Kubernetes)
func (d *dev) InstallerRestConfig(ctx context.Context) (*rest.Config, error) {
	backend, err := d.InstallerBackend(ctx)
	if err != nil {
		return nil, err
	}

	switch backend {
	case api.InstallerBackendPodman:
		return nil, fmt.Errorf("podman backend does not use Kubernetes")

	case api.InstallerBackendHive:
		// For Hive backend, use the Hive AKS cluster
		return d.HiveRestConfig(ctx, 0)

	case api.InstallerBackendAKSJob:
		// For dev, check if there's an override kubeconfig for installer execution
		installerKubeconfigPath := os.Getenv("INSTALLER_KUBE_CONFIG_PATH")
		if installerKubeconfigPath != "" {
			restConfig, err := clientcmd.BuildConfigFromFlags("", installerKubeconfigPath)
			if err != nil {
				return nil, err
			}
			return restConfig, nil
		}

		// Default to using the Hive AKS cluster for development
		// (CI will also use this - the retained Hive-host AKS cluster without Hive)
		return d.HiveRestConfig(ctx, 0)

	default:
		return nil, fmt.Errorf("unknown installer backend: %s", backend)
	}
}

func (p *prod) InstallerRestConfig(ctx context.Context) (*rest.Config, error) {
	backend, err := p.InstallerBackend(ctx)
	if err != nil {
		return nil, err
	}

	switch backend {
	case api.InstallerBackendPodman:
		return nil, fmt.Errorf("podman backend is not supported in production")

	case api.InstallerBackendHive:
		// For Hive backend, use the Hive AKS cluster
		return p.HiveRestConfig(ctx, 0)

	case api.InstallerBackendAKSJob:
		// TODO: This needs to be implemented based on the SVC cluster infrastructure
		// The SVC cluster naming pattern and discovery mechanism needs to be provided
		// by the infrastructure team (Workstream C: Service AKS and SDP-Pipelines)
		//
		// Expected implementation:
		// 1. Discover the regional SVC AKS cluster by location
		// 2. Use Microsoft Entra authentication with AKS Azure RBAC
		// 3. Obtain an AKS API token without persisting cluster-admin kubeconfig
		// 4. Return RestConfig with least-privilege access
		//
		// For now, return an error until the SVC cluster infrastructure is ready
		return nil, fmt.Errorf("SVC AKS cluster discovery not yet implemented - requires Workstream C infrastructure")

	default:
		return nil, fmt.Errorf("unknown installer backend: %s", backend)
	}
}
