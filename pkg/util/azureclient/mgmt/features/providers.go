package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
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
func NewProvidersClient(subscriptionID string, authorizer autorest.Authorizer) ProvidersClient {
	client := mgmtfeatures.NewProvidersClient(subscriptionID)
	client.Authorizer = authorizer

	return &providersClient{
		ProvidersClient: client,
	}
}
