package documentdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtdocumentdb "github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2019-08-01/documentdb"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// DatabaseAccountsClient is a minimal interface for azure DatabaseAccountsClient
type DatabaseAccountsClient interface {
	ListKeys(ctx context.Context, resourceGroupName string, accountName string) (result mgmtdocumentdb.DatabaseAccountListKeysResult, err error)
}

type databaseAccountsClient struct {
	mgmtdocumentdb.DatabaseAccountsClient
}

var _ DatabaseAccountsClient = &databaseAccountsClient{}

// NewDatabaseAccountsClient creates a new DatabaseAccountsClient
func NewDatabaseAccountsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) DatabaseAccountsClient {
	client := mgmtdocumentdb.NewDatabaseAccountsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &databaseAccountsClient{
		DatabaseAccountsClient: client,
	}
}
