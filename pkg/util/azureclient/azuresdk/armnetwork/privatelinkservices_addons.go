package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// PrivateLinkServicesClientAddons contains addons for PrivateLinkServicesClient
type PrivateLinkServicesClientAddons interface {
	List(ctx context.Context, resourceGroupName string, options *armnetwork.PrivateLinkServicesClientListOptions) ([]*armnetwork.PrivateLinkService, error)
	DeletePrivateEndpointConnectionAndWait(ctx context.Context, resourceGroupName string, serviceName string, peConnectionName string, options *armnetwork.PrivateLinkServicesClientBeginDeletePrivateEndpointConnectionOptions) error
}

func (c *privateLinkServicesClient) List(ctx context.Context, resourceGroupName string, options *armnetwork.PrivateLinkServicesClientListOptions) (result []*armnetwork.PrivateLinkService, err error) {
	pager := c.PrivateLinkServicesClient.NewListPager(resourceGroupName, options)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}
	return result, nil
}

func (c *privateLinkServicesClient) DeletePrivateEndpointConnectionAndWait(ctx context.Context, resourceGroupName string, serviceName string, peConnectionName string, options *armnetwork.PrivateLinkServicesClientBeginDeletePrivateEndpointConnectionOptions) error {
	poller, err := c.PrivateLinkServicesClient.BeginDeletePrivateEndpointConnection(ctx, resourceGroupName, serviceName, peConnectionName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
