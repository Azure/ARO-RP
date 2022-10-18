package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerservice"
)

type AksCredentialType int64

const (
	UserCredentials  AksCredentialType = 0
	AdminCredentials AksCredentialType = 1
)

type Manager interface {
	HiveRestConfig(context.Context, int, AksCredentialType) (*rest.Config, error)
	InstallViaHive(context.Context) (bool, error)
	AdoptByHive(context.Context) (bool, error)

	// Allows overriding the default installer pullspec for Prod, if the OpenShiftVersions database is not populated
	DefaultInstallerPullSpecOverride(context.Context) string
}

type dev struct {
	location              string
	managedClustersClient containerservice.ManagedClustersClient

	hiveCredentialsMutex *sync.RWMutex
	cachedCredentials    map[AksCredentialType]map[int]*rest.Config
}

func NewDev(location string, managedClustersClient containerservice.ManagedClustersClient) Manager {
	return &dev{location: location,
		managedClustersClient: managedClustersClient,
		cachedCredentials:     make(map[AksCredentialType]map[int]*rest.Config),
		hiveCredentialsMutex:  &sync.RWMutex{},
	}
}

type prod struct {
	location              string
	managedClustersClient containerservice.ManagedClustersClient

	hiveCredentialsMutex *sync.RWMutex
	cachedCredentials    map[AksCredentialType]map[int]*rest.Config
}

func NewProd(location string, managedClustersClient containerservice.ManagedClustersClient) Manager {
	return &prod{
		location:              location,
		managedClustersClient: managedClustersClient,
		cachedCredentials:     make(map[AksCredentialType]map[int]*rest.Config),
		hiveCredentialsMutex:  &sync.RWMutex{},
	}
}
