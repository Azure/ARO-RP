package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ManagedClustersClient is a minimal interface for azure ManagedClustersClient
type ManagedClustersClient interface {
	ManagedClustersAddons
}

type managedClustersClient struct {
	mgmtcontainerservice.ManagedClustersClient
}

var _ ManagedClustersClient = &managedClustersClient{}

// NewManagedClustersClient creates a new ManagedClustersClient
func NewManagedClustersClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) ManagedClustersClient {
	client := mgmtcontainerservice.NewManagedClustersClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &managedClustersClient{
		ManagedClustersClient: client,
	}
}
