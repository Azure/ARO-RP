package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

	"k8s.io/client-go/rest"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/Azure/ARO-RP/pkg/api"
	utilcontainerservice "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerservice"
)

const (
	hiveKubeconfigPathEnvVar  = "HIVE_KUBE_CONFIG_PATH"
	hiveInstallerEnableEnvVar = "ARO_INSTALL_VIA_HIVE"
	installerBackendEnvVar    = "ARO_INSTALLER_BACKEND"
	hiveDefaultPullSpecEnvVar = "ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC"
	hiveAdoptEnableEnvVar     = "ARO_ADOPT_BY_HIVE"
	installerAKSClusterEnvVar = "ARO_INSTALLER_AKS_CLUSTER_NAME"
	installerAKSRGEnvVar      = "ARO_INSTALLER_AKS_RESOURCE_GROUP"

	// installerHiveShard is the shard used when the installer falls back to the
	// Hive-host AKS cluster. Shard 0 resolves to aro-aks-cluster-000, which does
	// not exist, and to HIVE_KUBE_CONFIG_PATH, which CI does not set: CI exports
	// HIVE_KUBE_CONFIG_PATH_1 and the deployed cluster is aro-aks-cluster-001.
	// This matches the shard used by pkg/hive and pkg/backend.
	installerHiveShard = 1
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
	tokenCredential       azcore.TokenCredential

	hiveCredentialsMutex sync.RWMutex
	cachedCredentials    map[int]*rest.Config

	installerCredentialsMutex sync.RWMutex
	cachedInstallerConfig     *rest.Config
	cachedInstallerCluster    string
	cachedInstallerRG         string
}

func NewProd(location string, managedClustersClient utilcontainerservice.ManagedClustersClient, tokenCredential azcore.TokenCredential) Manager {
	return &prod{
		location:              location,
		managedClustersClient: managedClustersClient,
		tokenCredential:       tokenCredential,
		cachedCredentials:     make(map[int]*rest.Config),
		hiveCredentialsMutex:  sync.RWMutex{},
	}
}
