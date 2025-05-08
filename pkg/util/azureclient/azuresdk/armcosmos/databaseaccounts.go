package armcosmos

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdkcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
)

type DatabaseAccountsClient interface {
	ListKeys(ctx context.Context, resourceGroupName string, accountName string, options *sdkcosmos.DatabaseAccountsClientListKeysOptions) (sdkcosmos.DatabaseAccountsClientListKeysResponse, error)
}

type databaseAccountsClient struct {
	sdkcosmos.DatabaseAccountsClient
}

var _ DatabaseAccountsClient = &databaseAccountsClient{}

func NewDatabaseAccountsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (DatabaseAccountsClient, error) {
	return sdkcosmos.NewDatabaseAccountsClient(subscriptionID, credential, options)
}
