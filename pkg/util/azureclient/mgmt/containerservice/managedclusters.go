package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
)

// ManagedClustersClient is a minimal interface for azure ManagedClustersClient
type ManagedClustersClient interface {
	ManagedClustersAddons
}

type managedClustersClient struct {
	*armcontainerservice.ManagedClustersClient
}

var _ ManagedClustersClient = &managedClustersClient{}

// NewManagedClustersClient creates a new ManagedClustersClient
func NewManagedClustersClient(clientFactory *armcontainerservice.ClientFactory) ManagedClustersClient {
	client := clientFactory.NewManagedClustersClient()

	return &managedClustersClient{
		ManagedClustersClient: client,
	}
}
