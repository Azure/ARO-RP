package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerservice"
)

const (
	hiveKubeconfigPathEnvVar  = "HIVE_KUBE_CONFIG_PATH"
	hiveInstallerEnableEnvVar = "ARO_INSTALL_VIA_HIVE"
	hiveDefaultPullSpecEnvVar = "ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC"
	hiveAdoptEnableEnvVar     = "ARO_ADOPT_BY_HIVE"
	installStrategyEnvVar     = "ARO_INSTALL_STRATEGY"
)

type InstallStrategy = int

const (
	BuiltinStrategy InstallStrategy = iota
	HiveStrategy
	AKSStrategy
)

type Manager interface {
	HiveRestConfig(context.Context, int) (*rest.Config, error)
	InstallStrategy(context.Context) (InstallStrategy, error)
	AdoptByHive(context.Context) (bool, error)

	// Allows overriding the default installer pullspec for Prod, if the OpenShiftVersions database is not populated
	DefaultInstallerPullSpecOverride(context.Context) string
}

type dev struct {
	location              string
	managedClustersClient containerservice.ManagedClustersClient

	hiveCredentialsMutex sync.RWMutex
	cachedCredentials    map[int]*rest.Config
}

func NewDev(location string, managedClustersClient containerservice.ManagedClustersClient) Manager {
	return &dev{location: location,
		managedClustersClient: managedClustersClient,
		cachedCredentials:     make(map[int]*rest.Config),
		hiveCredentialsMutex:  sync.RWMutex{},
	}
}

type prod struct {
	location              string
	managedClustersClient containerservice.ManagedClustersClient

	hiveCredentialsMutex sync.RWMutex
	cachedCredentials    map[int]*rest.Config
}

func NewProd(location string, managedClustersClient containerservice.ManagedClustersClient) Manager {
	return &prod{
		location:              location,
		managedClustersClient: managedClustersClient,
		cachedCredentials:     make(map[int]*rest.Config),
		hiveCredentialsMutex:  sync.RWMutex{},
	}
}
