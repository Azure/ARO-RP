package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// UsageClient is a minimal interface for azure UsageClient
type UsageClient interface {
	UsageClientAddons
}

type usageClient struct {
	mgmtcompute.UsageClient
}

var _ UsageClient = &usageClient{}

// NewUsageClient creates a new UsageClient
func NewUsageClient(environment *azureclient.AROEnvironment, tenantID string, authorizer autorest.Authorizer) UsageClient {
	client := mgmtcompute.NewUsageClientWithBaseURI(environment.ResourceManagerEndpoint, tenantID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &usageClient{
		UsageClient: client,
	}
}
