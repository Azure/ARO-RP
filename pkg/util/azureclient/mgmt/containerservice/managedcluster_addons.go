package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
)

// RegistriesAddons contains addons for RegistriesClient
type ManagedClustersAddons interface {
	ListClusterUserCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (mgmtcontainerservice.CredentialResults, error)
}

func (r *managedClustersClient) ListClusterUserCredentials(ctx context.Context, resourceGroupName string, resourceName string, serverFqdn string) (mgmtcontainerservice.CredentialResults, error) {
	return r.ManagedClustersClient.ListClusterUserCredentials(ctx, resourceGroupName, resourceName, serverFqdn)
}
