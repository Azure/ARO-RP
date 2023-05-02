package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ResourcesClient is a minimal interface for azure ResourcesClient
type ResourcesClient interface {
	GetByID(ctx context.Context, resourceID string, APIVersion string) (mgmtfeatures.GenericResource, error)
	DeleteByID(ctx context.Context, resourceID string, APIVersion string) (mgmtfeatures.ResourcesDeleteByIDFuture, error)
	ResourcesClientAddons
}

type resourcesClient struct {
	mgmtfeatures.ResourcesClient
}

var _ ResourcesClient = &resourcesClient{}

// NewResourcesClient creates a new ResourcesClient
func NewResourcesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) ResourcesClient {
	client := mgmtfeatures.NewResourcesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &resourcesClient{
		ResourcesClient: client,
	}
}
