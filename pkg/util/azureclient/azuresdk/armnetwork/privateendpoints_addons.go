package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

// PrivateEndpointsClientAddons contains addons for PrivateEndpointsClient
type PrivateEndpointsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, privateEndpointName string, parameters armnetwork.PrivateEndpoint, options *armnetwork.PrivateEndpointsClientBeginCreateOrUpdateOptions) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string, options *armnetwork.PrivateEndpointsClientBeginDeleteOptions) error
}

func (c *privateEndpointsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, privateEndpointName string, parameters armnetwork.PrivateEndpoint, options *armnetwork.PrivateEndpointsClientBeginCreateOrUpdateOptions) error {
	poller, err := c.BeginCreateOrUpdate(ctx, resourceGroupName, privateEndpointName, parameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *privateEndpointsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string, options *armnetwork.PrivateEndpointsClientBeginDeleteOptions) error {
	poller, err := c.BeginDelete(ctx, resourceGroupName, publicIPAddressName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
