package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// ManagedClustersAddons is a minimal interface for azure ManagedClustersAddons
type ManagedClustersAddons interface {
	ListClusterAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse, error)
	List(ctx context.Context) *runtime.Pager[armcontainerservice.ManagedClustersClientListResponse]
}

func (r *managedClustersClient) ListClusterAdminCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (armcontainerservice.ManagedClustersClientListClusterAdminCredentialsResponse, error) {
	return r.ManagedClustersClient.ListClusterAdminCredentials(ctx, resourceGroupName, resourceName, &armcontainerservice.ManagedClustersClientListClusterAdminCredentialsOptions{
		ServerFqdn: pointerutils.ToPtr(serverFqdn),
	})
}

func (r *managedClustersClient) List(ctx context.Context) *runtime.Pager[armcontainerservice.ManagedClustersClientListResponse] {
	return r.NewListPager(nil)
}
