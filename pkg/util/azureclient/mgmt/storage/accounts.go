package storage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
)

// AccountsClient is a minimal interface for azure AccountsClient
type AccountsClient interface {
	ListAccountSAS(ctx context.Context, resourceGroupName string, accountName string, parameters mgmtstorage.AccountSasParameters) (result mgmtstorage.ListAccountSasResponse, err error)
	ListByResourceGroup(context context.Context, resourceGroup string) (mgmtstorage.AccountListResult, error)
	ListKeys(ctx context.Context, resourceGroupName string, accountName string, expand mgmtstorage.ListKeyExpand) (result mgmtstorage.AccountListKeysResult, err error)
	AccountsClientAddons
}

type accountsClient struct {
	mgmtstorage.AccountsClient
}

var _ AccountsClient = &accountsClient{}

// NewAccountsClient returns a new AccountsClient
func NewAccountsClient(subscriptionID string, authorizer autorest.Authorizer) AccountsClient {
	client := mgmtstorage.NewAccountsClient(subscriptionID)
	client.Authorizer = authorizer

	return &accountsClient{
		AccountsClient: client,
	}
}
