package privatedns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
)

// VirtualNetworkLinksClientAddons contains addons for VirtualNetworkLinksClient
type VirtualNetworkLinksClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string, privateZoneName string, virtualNetworkLinkName string, ifMatch string) error
	List(ctx context.Context, resourceGroupName string, privateZoneName string, top *int32) ([]mgmtprivatedns.VirtualNetworkLink, error)
}

func (c *virtualNetworkLinksClient) DeleteAndWait(ctx context.Context, resourceGroupName string, privateZoneName string, virtualNetworkLinkName string, ifMatch string) error {
	future, err := c.Delete(ctx, resourceGroupName, privateZoneName, virtualNetworkLinkName, ifMatch)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualNetworkLinksClient) List(ctx context.Context, resourceGroupName string, privateZoneName string, top *int32) (virtualNetworkLinks []mgmtprivatedns.VirtualNetworkLink, err error) {
	page, err := c.VirtualNetworkLinksClient.List(ctx, resourceGroupName, privateZoneName, top)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		virtualNetworkLinks = append(virtualNetworkLinks, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return virtualNetworkLinks, nil
}
