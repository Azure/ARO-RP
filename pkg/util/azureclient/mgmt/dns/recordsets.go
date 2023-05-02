package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// RecordSetsClient is a minimal interface for azure RecordSetsClient
type RecordSetsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType mgmtdns.RecordType, parameters mgmtdns.RecordSet, ifMatch string, ifNoneMatch string) (result mgmtdns.RecordSet, err error)
	Delete(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType mgmtdns.RecordType, ifMatch string) (result autorest.Response, err error)
	Get(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType mgmtdns.RecordType) (result mgmtdns.RecordSet, err error)
}

type recordSetsClient struct {
	mgmtdns.RecordSetsClient
}

var _ RecordSetsClient = &recordSetsClient{}

// NewRecordSetsClient creates a new RecordSetsClient
func NewRecordSetsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) RecordSetsClient {
	client := mgmtdns.NewRecordSetsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &recordSetsClient{
		RecordSetsClient: client,
	}
}
