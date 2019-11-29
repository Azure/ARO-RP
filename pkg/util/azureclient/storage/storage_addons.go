package storage

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
)

type AccountsClientAddons interface {
	Create(ctx context.Context, resourceGroupName string, accountName string, parameters storage.AccountCreateParameters) error
}

func (c *accountsClient) Create(ctx context.Context, resourceGroupName string, accountName string, parameters storage.AccountCreateParameters) error {
	future, err := c.AccountsClient.Create(ctx, resourceGroupName, accountName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.AccountsClient.Client)
}
