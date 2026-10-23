package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
)

// ManagedClustersAddons is a minimal interface for azure ManagedClustersAddons
type ManagedClustersAddons interface {
	ListClusterAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse, error)
	ListClusterUserCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (armcontainerservice.ManagedClustersClientListClusterUserCredentialsResponse, error)
	List(ctx context.Context) *runtime.Pager[armcontainerservice.ManagedClustersClientListResponse]
	Get(ctx context.Context, resourceGroupName string, resourceName string) (armcontainerservice.ManagedClustersClientGetResponse, error)
}

func (r *managedClustersClient) Get(ctx context.Context, resourceGroupName string, resourceName string) (armcontainerservice.ManagedClustersClientGetResponse, error) {
	return r.ManagedClustersClient.Get(ctx, resourceGroupName, resourceName, nil)
}

func (r *managedClustersClient) ListClusterAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse, error) {
	return r.ManagedClustersClient.ListClusterAdminCredentials(ctx, resourceGroupName, resourceName, &armcontainerservice.ManagedClustersClientListClusterAdminCredentialsOptions{
		ServerFqdn: new(serverFqdn),
	})
}

func (r *managedClustersClient) ListClusterUserCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (armcontainerservice.ManagedClustersClientListClusterUserCredentialsResponse, error) {
	options := &armcontainerservice.ManagedClustersClientListClusterUserCredentialsOptions{}
	if serverFqdn != "" {
		options.ServerFqdn = new(serverFqdn)
	}
	return r.ManagedClustersClient.ListClusterUserCredentials(ctx, resourceGroupName, resourceName, options)
}

func (r *managedClustersClient) List(ctx context.Context) *runtime.Pager[armcontainerservice.ManagedClustersClientListResponse] {
	return r.NewListPager(nil)
}
