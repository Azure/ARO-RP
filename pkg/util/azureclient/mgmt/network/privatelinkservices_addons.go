package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
)

// PrivateLinkServicesClientAddons contains addons for PrivateLinkServicesClient
type PrivateLinkServicesClientAddons interface {
	List(ctx context.Context, resourceGroupName string) (privatelinkservices []mgmtnetwork.PrivateLinkService, err error)
}

func (c *privateLinkServicesClient) List(ctx context.Context, resourceGroupName string) (privatelinkservices []mgmtnetwork.PrivateLinkService, err error) {
	page, err := c.PrivateLinkServicesClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		privatelinkservices = append(privatelinkservices, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return privatelinkservices, nil
}
