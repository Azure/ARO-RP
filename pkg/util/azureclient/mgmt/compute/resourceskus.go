package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ResourceSkusClient is a minimal interface for azure ResourceSkusClient
type ResourceSkusClient interface {
	ResourceSkusClientAddons
}

type resourceSkusClient struct {
	mgmtcompute.ResourceSkusClient
}

var _ ResourceSkusClient = &resourceSkusClient{}

// NewResourceSkusClient creates a new ResourceSkusClient
func NewResourceSkusClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) ResourceSkusClient {
	client := mgmtcompute.NewResourceSkusClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &resourceSkusClient{
		ResourceSkusClient: client,
	}
}
