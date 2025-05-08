package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// PublicIPAddressesClientAddons contains addons for PublicIPAddressesClient
type PublicIPAddressesClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string, parameters armnetwork.PublicIPAddress, options *armnetwork.PublicIPAddressesClientBeginCreateOrUpdateOptions) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string, options *armnetwork.PublicIPAddressesClientBeginDeleteOptions) (err error)
	List(ctx context.Context, resourceGroupName string, options *armnetwork.PublicIPAddressesClientListOptions) (result []*armnetwork.PublicIPAddress, err error)
}

func (c *publicIPAddressesClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string, parameters armnetwork.PublicIPAddress, options *armnetwork.PublicIPAddressesClientBeginCreateOrUpdateOptions) error {
	poller, err := c.BeginCreateOrUpdate(ctx, resourceGroupName, publicIPAddressName, parameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *publicIPAddressesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, publicIPAddressName string, options *armnetwork.PublicIPAddressesClientBeginDeleteOptions) error {
	poller, err := c.BeginDelete(ctx, resourceGroupName, publicIPAddressName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *publicIPAddressesClient) List(ctx context.Context, resourceGroupName string, options *armnetwork.PublicIPAddressesClientListOptions) (result []*armnetwork.PublicIPAddress, err error) {
	pager := c.PublicIPAddressesClient.NewListPager(resourceGroupName, options)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}
	return result, nil
}
