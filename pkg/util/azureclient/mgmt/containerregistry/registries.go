package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// RegistriesClient is a minimal interface for azure RegistriesClient
type RegistriesClient interface {
	RegistriesAddons
}

type registriesClient struct {
	mgmtcontainerregistry.RegistriesClient
}

var _ RegistriesClient = &registriesClient{}

// NewRegistriesClient creates a new RegistriesClient
func NewRegistriesClient(environment *azure.Environment, subscriptionID string, authorizer autorest.Authorizer) RegistriesClient {
	client := mgmtcontainerregistry.NewRegistriesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &registriesClient{
		RegistriesClient: client,
	}
}
