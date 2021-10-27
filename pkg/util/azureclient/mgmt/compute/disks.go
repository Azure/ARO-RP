package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// DisksClient is a minimal interface for azure DisksClient
type DisksClient interface {
	Get(ctx context.Context, resourceGroupName string, diskName string) (result mgmtcompute.Disk, err error)
	DisksClientAddons
}

type disksClient struct {
	mgmtcompute.DisksClient
}

var _ DisksClient = &disksClient{}

// NewDisksClient creates a new DisksClient
func NewDisksClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) DisksClient {
	client := mgmtcompute.NewDisksClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &disksClient{
		DisksClient: client,
	}
}
