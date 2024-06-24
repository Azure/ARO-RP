package armdns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdkdns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
)

// RecordSetsClient is a minimal interface for azure RecordSetsClient
type RecordSetsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType sdkdns.RecordType, parameters sdkdns.RecordSet, options *sdkdns.RecordSetsClientCreateOrUpdateOptions) (sdkdns.RecordSetsClientCreateOrUpdateResponse, error)
	Delete(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType sdkdns.RecordType, options *sdkdns.RecordSetsClientDeleteOptions) (sdkdns.RecordSetsClientDeleteResponse, error)
	Get(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType sdkdns.RecordType, options *sdkdns.RecordSetsClientGetOptions) (sdkdns.RecordSetsClientGetResponse, error)
}

type recordSetsClient struct {
	sdkdns.RecordSetsClient
}

var _ RecordSetsClient = &recordSetsClient{}

// NewRecordSetsClient creates a new RecordSetsClient
func NewRecordSetsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) RecordSetsClient {
	clientFactory, err := sdkdns.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil
	}
	client := clientFactory.NewRecordSetsClient()
	return &recordSetsClient{
		RecordSetsClient: *client,
	}
}
