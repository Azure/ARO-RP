package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2015-12-01/features"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// Client is a minimal interface for azure Client
type Client interface {
	Get(ctx context.Context, resourceProviderNamespace string, featureName string) (result mgmtfeatures.Result, err error)
	Register(ctx context.Context, resourceProviderNamespace string, featureName string) (result mgmtfeatures.Result, err error)
}

type client struct {
	mgmtfeatures.Client
}

var _ Client = &client{}

// NewClient creates a new Client
func NewClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) Client {
	_client := mgmtfeatures.NewClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	_client.Authorizer = authorizer

	return &client{
		Client: _client,
	}
}
