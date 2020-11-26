package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// DisksClient is a minimal interface for azure DisksClient
type DisksClient interface {
	DisksClientAddons
}

type disksClient struct {
	mgmtcompute.DisksClient
}

var _ DisksClient = &disksClient{}

// NewDisksClient creates a new DisksClient
func NewDisksClient(environment *azure.Environment, subscriptionID string, authorizer autorest.Authorizer) DisksClient {
	client := mgmtcompute.NewDisksClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &disksClient{
		DisksClient: client,
	}
}
