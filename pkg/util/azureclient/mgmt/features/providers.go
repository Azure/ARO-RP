package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ProvidersClient is a minimal interface for azure ProvidersClient
type ProvidersClient interface {
	Get(ctx context.Context, resourceProviderNamespace string, expand string) (result mgmtresources.Provider, err error)
	ProvidersClientAddons
}

type providersClient struct {
	mgmtresources.ProvidersClient
}

var _ ProvidersClient = &providersClient{}

// NewProvidersClient creates a new ProvidersClient
func NewProvidersClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) ProvidersClient {
	client := mgmtresources.NewProvidersClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &providersClient{
		ProvidersClient: client,
	}
}
