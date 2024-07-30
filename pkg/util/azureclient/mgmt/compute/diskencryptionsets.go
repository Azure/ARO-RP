package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// DiskEncryptionSetsClient is a minimal interface for azure DiskEncryptionSetsClient
type DiskEncryptionSetsClient interface {
	Get(ctx context.Context, resourceGroupName string, diskEncryptionSetName string) (result mgmtcompute.DiskEncryptionSet, err error)
}

type diskEncryptionSetsClient struct {
	mgmtcompute.DiskEncryptionSetsClient
}

var _ DiskEncryptionSetsClient = &diskEncryptionSetsClient{}

// NewDisksClient creates a new DisksClient
func NewDiskEncryptionSetsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) DiskEncryptionSetsClient {
	client := mgmtcompute.NewDiskEncryptionSetsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &diskEncryptionSetsClient{
		DiskEncryptionSetsClient: client,
	}
}
