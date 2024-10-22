package armstorage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

// AccountsClient is a minimal interface for Azure AccountsClient
type AccountsClient interface {
	GetProperties(ctx context.Context, resourceGroupName string, accountName string, options *armstorage.AccountsClientGetPropertiesOptions) (armstorage.AccountsClientGetPropertiesResponse, error)
	ListAccountSAS(ctx context.Context, resourceGroupName string, accountName string, parameters armstorage.AccountSasParameters, options *armstorage.AccountsClientListAccountSASOptions) (armstorage.AccountsClientListAccountSASResponse, error)
}

type accountsClient struct {
	*armstorage.AccountsClient
}

var _ AccountsClient = &accountsClient{}

// NewAccountsClient creates a new AccountsClient
func NewAccountsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (AccountsClient, error) {
	clientFactory, err := armstorage.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &accountsClient{AccountsClient: clientFactory.NewAccountsClient()}, nil
}
