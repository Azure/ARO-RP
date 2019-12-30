package documentdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/$GOPACKAGE DatabaseAccountsClient
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2015-04-08/documentdb"
	"github.com/Azure/go-autorest/autorest"
)

// DatabaseAccountsClient is a minimal interface for azure DatabaseAccountsClient
type DatabaseAccountsClient interface {
	ListByResourceGroup(ctx context.Context, resourceGroupName string) (result documentdb.DatabaseAccountsListResult, err error)
	ListKeys(ctx context.Context, resourceGroupName string, accountName string) (result documentdb.DatabaseAccountListKeysResult, err error)
}

type databaseAccountsClient struct {
	documentdb.DatabaseAccountsClient
}

var _ DatabaseAccountsClient = &databaseAccountsClient{}

// NewDatabaseAccountsClient creates a new DatabaseAccountsClient
func NewDatabaseAccountsClient(subscriptionID string, authorizer autorest.Authorizer) DatabaseAccountsClient {
	client := documentdb.NewDatabaseAccountsClient(subscriptionID)
	client.Authorizer = authorizer

	return &databaseAccountsClient{
		DatabaseAccountsClient: client,
	}
}
