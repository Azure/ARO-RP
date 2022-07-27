package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
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
	location        string
	managedClusters containerservice.ManagedClustersClient

	cachedCredentials map[int][]mgmtcontainerservice.CredentialResult
}

func NewProd(location string, managedClusters containerservice.ManagedClustersClient) Manager {
	return &prod{
		location:          location,
		managedClusters:   managedClusters,
		cachedCredentials: make(map[int][]mgmtcontainerservice.CredentialResult),
	}
}
