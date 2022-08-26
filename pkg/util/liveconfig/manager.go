package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerservice"
)

type Manager interface {
	HiveRestConfig(context.Context, int) (*rest.Config, error)
}

type dev struct{}

func NewDev() Manager {
	return &dev{}
}

type prod struct {
	location              string
	managedClustersClient containerservice.ManagedClustersClient

	hiveCredentialsMutex *sync.RWMutex
	cachedCredentials    map[int]*rest.Config
}

func NewProd(location string, managedClustersClient containerservice.ManagedClustersClient) Manager {
	return &prod{
		location:              location,
		managedClustersClient: managedClustersClient,
		cachedCredentials:     make(map[int]*rest.Config),
		hiveCredentialsMutex:  &sync.RWMutex{},
	}
}
