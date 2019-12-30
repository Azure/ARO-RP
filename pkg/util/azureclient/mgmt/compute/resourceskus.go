package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

// ResourceSkusClient is a minimal interface for azure ResourceSkusClient
type ResourceSkusClient interface {
	ResourceSkusClientAddons
}

type resourceSkusClient struct {
	compute.ResourceSkusClient
}

var _ ResourceSkusClient = &resourceSkusClient{}

// NewResourceSkusClient creates a new ResourceSkusClient
func NewResourceSkusClient(subscriptionID string, authorizer autorest.Authorizer) ResourceSkusClient {
	client := compute.NewResourceSkusClient(subscriptionID)
	client.Authorizer = authorizer

	return &resourceSkusClient{
		ResourceSkusClient: client,
	}
}
