package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

// DisksClient is a minimal interface for azure DisksClient
type DisksClient interface {
	DisksClientAddons
}

type disksClient struct {
	compute.DisksClient
}

var _ DisksClient = &disksClient{}

// NewDisksClient creates a new DisksClient
func NewDisksClient(subscriptionID string, authorizer autorest.Authorizer) DisksClient {
	client := compute.NewDisksClient(subscriptionID)
	client.Authorizer = authorizer

	return &disksClient{
		DisksClient: client,
	}
}
