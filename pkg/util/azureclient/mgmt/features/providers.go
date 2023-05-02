package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ProvidersClient is a minimal interface for azure ProvidersClient
type ProvidersClient interface {
	ProvidersClientAddons
}

type providersClient struct {
	mgmtfeatures.ProvidersClient
}

var _ ProvidersClient = &providersClient{}

// NewProvidersClient creates a new ProvidersClient
func NewProvidersClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) ProvidersClient {
	client := mgmtfeatures.NewProvidersClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &providersClient{
		ProvidersClient: client,
	}
}
