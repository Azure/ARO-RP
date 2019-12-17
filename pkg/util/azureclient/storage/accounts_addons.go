package storage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
)

// AccountsClientAddons contains addons for AccountsClient
type AccountsClientAddons interface {
	CreateAndWait(ctx context.Context, resourceGroupName string, accountName string, parameters storage.AccountCreateParameters) (err error)
}

func (c *accountsClient) CreateAndWait(ctx context.Context, resourceGroupName string, accountName string, parameters storage.AccountCreateParameters) error {
	future, err := c.Create(ctx, resourceGroupName, accountName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.AccountsClient.Client)
}
