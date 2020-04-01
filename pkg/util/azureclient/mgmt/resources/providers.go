package resources

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

// ProvidersClient is a minimal interface for azure ProvidersClient
type ProvidersClient interface {
	ProvidersClientAddons
}

type providersClient struct {
	mgmtresources.ProvidersClient
}

var _ ProvidersClient = &providersClient{}

// NewProvidersClient creates a new ProvidersClient
func NewProvidersClient(subscriptionID string, authorizer autorest.Authorizer) ProvidersClient {
	client := mgmtresources.NewProvidersClient(subscriptionID)
	client.Authorizer = authorizer

	return &providersClient{
		ProvidersClient: client,
	}
}
