package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

// SecurityGroupsClientAddons contains addons for SecurityGroupsClient
type SecurityGroupsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, parameters mgmtnetwork.SecurityGroup) (err error)
	DeleteAndWait(ctx context.Context, resourceGroupName string, networkSecurityGroupName string) (err error)
	List(ctx context.Context, resourceGroupName string) (result []mgmtnetwork.SecurityGroup, err error)
}

func (c *securityGroupsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, parameters mgmtnetwork.SecurityGroup) (err error) {
	future, err := c.SecurityGroupsClient.CreateOrUpdate(ctx, resourceGroupName, networkSecurityGroupName, parameters)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *securityGroupsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, networkSecurityGroupName string) (err error) {
	future, err := c.SecurityGroupsClient.Delete(ctx, resourceGroupName, networkSecurityGroupName)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *securityGroupsClient) List(ctx context.Context, resourceGroupName string) (result []mgmtnetwork.SecurityGroup, err error) {
	page, err := c.SecurityGroupsClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		result = append(result, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
