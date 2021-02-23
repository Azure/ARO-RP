package privatedns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
)

// PrivateZonesClientAddons contains addons for PrivateZonesClient
type PrivateZonesClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string, privateZoneName string, ifMatch string) error
	ListByResourceGroup(ctx context.Context, resourceGroupName string, top *int32) ([]mgmtprivatedns.PrivateZone, error)
}

func (c *privateZonesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, privateZoneName string, ifMatch string) error {
	future, err := c.PrivateZonesClient.Delete(ctx, resourceGroupName, privateZoneName, ifMatch)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *privateZonesClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, top *int32) (privateZones []mgmtprivatedns.PrivateZone, err error) {
	page, err := c.PrivateZonesClient.ListByResourceGroup(ctx, resourceGroupName, top)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		privateZones = append(privateZones, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return privateZones, nil
}
