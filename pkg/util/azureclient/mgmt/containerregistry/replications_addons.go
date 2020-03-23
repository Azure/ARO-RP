package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
)

// ReplicationsAddons contains addons for ReplicationsClient
type ReplicationsAddons interface {
	CreateAndWait(ctx context.Context, resourceGroupName string, registryName string, replicationName string, replication mgmtcontainerregistry.Replication) (err error)
}

func (r *replicationsClient) CreateAndWait(ctx context.Context, resourceGroupName string, registryName string, replicationName string, replication mgmtcontainerregistry.Replication) (err error) {
	future, err := r.ReplicationsClient.Create(ctx, resourceGroupName, registryName, replicationName, replication)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, r.Client)
}
