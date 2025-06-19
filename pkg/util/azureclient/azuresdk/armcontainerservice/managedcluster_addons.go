package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
)

// ManagedClustersAddons is a minimal interface for azure ManagedClustersAddons
type ManagedClustersAddons interface {
	ListClusterAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse, error)
	List(ctx context.Context) *runtime.Pager[armcontainerservice.ManagedClustersClientListResponse]
}

func (r *managedClustersClient) ListClusterAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse, error) {
	return r.ManagedClustersClient.ListClusterAdminCredentials(ctx, resourceGroupName, resourceName, &armcontainerservice.ManagedClustersClientListClusterAdminCredentialsOptions{
		ServerFqdn: to.Ptr(serverFqdn),
	})
}

func (r *managedClustersClient) List(ctx context.Context) *runtime.Pager[armcontainerservice.ManagedClustersClientListResponse] {
	return r.NewListPager(nil)
}
