package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
)

// ZonesClientAddons contains addons for ZonesClient
type ZonesClientAddons interface {
	ListByResourceGroup(ctx context.Context, resourceGroupName string, top *int32) (zones []dns.Zone, err error)
}

func (c *zonesClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, top *int32) (zones []dns.Zone, err error) {
	page, err := c.ZonesClient.ListByResourceGroup(ctx, resourceGroupName, top)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		zones = append(zones, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return zones, nil
}
