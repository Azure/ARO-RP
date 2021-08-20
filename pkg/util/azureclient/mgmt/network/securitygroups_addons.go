package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
)

// SecurityGroupsClientAddons contains addons for SecurityGroupsClient
type SecurityGroupsClientAddons interface {
	List(ctx context.Context, resourceGroupName string) (result []mgmtnetwork.SecurityGroup, err error)
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
