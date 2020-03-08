package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
)

// VaultsClientAddons contains addons for VaultsClient
type VaultsClientAddons interface {
	ListByResourceGroup(ctx context.Context, resourceGroupName string, top *int32) (vaults []keyvault.Vault, err error)
}

func (c *vaultsClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, top *int32) (vaults []keyvault.Vault, err error) {
	page, err := c.VaultsClient.ListByResourceGroup(ctx, resourceGroupName, top)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		vaults = append(vaults, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return vaults, nil
}
