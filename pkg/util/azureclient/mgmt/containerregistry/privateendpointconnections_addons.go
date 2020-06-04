package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2019-12-01-preview/containerregistry"
)

// PrivateEndpointConnectionsClientAddons contains addons for PrivateEndpointConnectionsClientAddons
type PrivateEndpointConnectionsClientAddons interface {
	List(ctx context.Context, resourceGroupName string, registryName string) (result []mgmtcontainerregistry.PrivateEndpointConnection, err error)
}

func (r *privateEndpointConnectionsClient) List(ctx context.Context, resourceGroupName string, registryName string) (result []mgmtcontainerregistry.PrivateEndpointConnection, err error) {
	itr, err := r.PrivateEndpointConnectionsClient.ListComplete(ctx, resourceGroupName, registryName)
	if err != nil {
		return nil, err
	}

	for itr.NotDone() {
		result = append(result, itr.Value())

		err = itr.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
