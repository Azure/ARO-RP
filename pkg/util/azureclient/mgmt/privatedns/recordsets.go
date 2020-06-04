package privatedns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest"
)

// RecordSetsClient is a minimal interface for azure RecordSetsClient
type RecordSetsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, privateZoneName string, recordType mgmtprivatedns.RecordType, relativeRecordSetName string, parameters mgmtprivatedns.RecordSet, ifMatch string, ifNoneMatch string) (result mgmtprivatedns.RecordSet, err error)
	Get(ctx context.Context, resourceGroupName string, privateZoneName string, recordType mgmtprivatedns.RecordType, relativeRecordSetName string) (result mgmtprivatedns.RecordSet, err error)
}

type recordSetsClient struct {
	mgmtprivatedns.RecordSetsClient
}

var _ RecordSetsClient = &recordSetsClient{}

// NewRecordSetsClient creates a new RecordSetsClient
func NewRecordSetsClient(subscriptionID string, authorizer autorest.Authorizer) RecordSetsClient {
	client := mgmtprivatedns.NewRecordSetsClient(subscriptionID)
	client.Authorizer = authorizer

	return &recordSetsClient{
		RecordSetsClient: client,
	}
}
