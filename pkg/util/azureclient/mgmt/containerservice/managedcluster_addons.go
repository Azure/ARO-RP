package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
)

// ManagedClustersAddons is a minimal interface for azure ManagedClustersAddons
type ManagedClustersAddons interface {
	ListClusterAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (mgmtcontainerservice.CredentialResults, error)
	List(ctx context.Context) (mgmtcontainerservice.ManagedClusterListResultPage, error)
}

func (r *managedClustersClient) ListClusterAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (mgmtcontainerservice.CredentialResults, error) {
	return r.ManagedClustersClient.ListClusterAdminCredentials(ctx, resourceGroupName, resourceName, serverFqdn)
}

func (r *managedClustersClient) List(ctx context.Context) (mgmtcontainerservice.ManagedClusterListResultPage, error) {
	return r.ManagedClustersClient.List(ctx)
}
