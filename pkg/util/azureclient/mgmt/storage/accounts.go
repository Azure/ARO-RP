package storage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// AccountsClient is a minimal interface for azure AccountsClient
type AccountsClient interface {
	GetProperties(ctx context.Context, resourceGroupName string, accountName string, expand mgmtstorage.AccountExpand) (result mgmtstorage.Account, err error)
	Update(ctx context.Context, resourceGroupName string, accountName string, parameters mgmtstorage.AccountUpdateParameters) (result mgmtstorage.Account, err error)
	ListAccountSAS(ctx context.Context, resourceGroupName string, accountName string, parameters mgmtstorage.AccountSasParameters) (result mgmtstorage.ListAccountSasResponse, err error)
	ListKeys(ctx context.Context, resourceGroupName string, accountName string, expand mgmtstorage.ListKeyExpand) (result mgmtstorage.AccountListKeysResult, err error)
}

type accountsClient struct {
	mgmtstorage.AccountsClient
}

var _ AccountsClient = &accountsClient{}

// NewAccountsClient returns a new AccountsClient
func NewAccountsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) AccountsClient {
	client := mgmtstorage.NewAccountsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &accountsClient{
		AccountsClient: client,
	}
}
