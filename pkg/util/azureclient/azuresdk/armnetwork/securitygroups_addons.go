package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// SecurityGroupsClientAddons contains addons for SecurityGroupsClient
type SecurityGroupsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, parameters armnetwork.SecurityGroup, options *armnetwork.SecurityGroupsClientBeginCreateOrUpdateOptions) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityGroupsClientBeginDeleteOptions) error
	List(ctx context.Context, resourceGroupName string, options *armnetwork.SecurityGroupsClientListOptions) ([]*armnetwork.SecurityGroup, error)
}

func (c *securityGroupsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, parameters armnetwork.SecurityGroup, options *armnetwork.SecurityGroupsClientBeginCreateOrUpdateOptions) error {
	poller, err := c.SecurityGroupsClient.BeginCreateOrUpdate(ctx, resourceGroupName, networkSecurityGroupName, parameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *securityGroupsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityGroupsClientBeginDeleteOptions) error {
	poller, err := c.SecurityGroupsClient.BeginDelete(ctx, resourceGroupName, networkSecurityGroupName, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *securityGroupsClient) List(ctx context.Context, resourceGroupName string, options *armnetwork.SecurityGroupsClientListOptions) (result []*armnetwork.SecurityGroup, err error) {
	pager := c.SecurityGroupsClient.NewListPager(resourceGroupName, options)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}
	return result, nil
}
