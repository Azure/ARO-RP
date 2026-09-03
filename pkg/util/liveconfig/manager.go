package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	utilcontainerservice "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerservice"
)

const (
	hiveKubeconfigPathEnvVar  = "HIVE_KUBE_CONFIG_PATH"
	hiveInstallerEnableEnvVar = "ARO_INSTALL_VIA_HIVE"
	installerBackendEnvVar    = "ARO_INSTALLER_BACKEND"
	hiveDefaultPullSpecEnvVar = "ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC"
	hiveAdoptEnableEnvVar     = "ARO_ADOPT_BY_HIVE"
)

type Manager interface {
	HiveRestConfig(context.Context, int) (*rest.Config, error)

	// InstallerBackend returns the configured installer backend (Hive, AKSJob, Podman)
	InstallerBackend(context.Context) (api.InstallerBackend, error)

	// InstallerRestConfig returns the Kubernetes RestConfig for the installer execution cluster
	// based on the selected backend. For AKSJob, returns the regional SVC AKS cluster in production
	// or the retained Hive-host AKS cluster in CI. For Hive, returns the Hive AKS cluster.
	InstallerRestConfig(context.Context) (*rest.Config, error)

	// InstallViaHive is deprecated - use InstallerBackend instead
	// Kept for backward compatibility during migration
	InstallViaHive(context.Context) (bool, error)

	AdoptByHive(context.Context) (bool, error)

	// Allows overriding the default installer pullspec for Prod, if the OpenShiftVersions database is not populated
	DefaultInstallerPullSpecOverride(context.Context) string
}

type dev struct {
	location              string
	managedClustersClient utilcontainerservice.ManagedClustersClient

	hiveCredentialsMutex sync.RWMutex
	cachedCredentials    map[int]*rest.Config
}

func NewDev(location string, managedClustersClient utilcontainerservice.ManagedClustersClient) Manager {
	return &dev{
		location:              location,
		managedClustersClient: managedClustersClient,
		cachedCredentials:     make(map[int]*rest.Config),
		hiveCredentialsMutex:  sync.RWMutex{},
	}
}

type prod struct {
	location              string
	managedClustersClient utilcontainerservice.ManagedClustersClient

	hiveCredentialsMutex sync.RWMutex
	cachedCredentials    map[int]*rest.Config
}

func NewProd(location string, managedClustersClient utilcontainerservice.ManagedClustersClient) Manager {
	return &prod{
		location:              location,
		managedClustersClient: managedClustersClient,
		cachedCredentials:     make(map[int]*rest.Config),
		hiveCredentialsMutex:  sync.RWMutex{},
	}
}
