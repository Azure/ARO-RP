package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
)

// ResourcesClient is a minimal interface for azure ResourcesClient
type ResourcesClient interface {
	GetByID(ctx context.Context, resourceID string, APIVersion string) (mgmtfeatures.GenericResource, error)
	ResourcesClientAddons
}

type resourcesClient struct {
	mgmtfeatures.ResourcesClient
}

var _ ResourcesClient = &resourcesClient{}

// NewResourcesClient creates a new ResourcesClient
func NewResourcesClient(subscriptionID string, authorizer autorest.Authorizer) ResourcesClient {
	client := mgmtfeatures.NewResourcesClient(subscriptionID)
	client.Authorizer = authorizer

	return &resourcesClient{
		ResourcesClient: client,
	}
}
