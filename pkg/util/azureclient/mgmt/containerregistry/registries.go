package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
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
func NewRegistriesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) RegistriesClient {
	client := mgmtcontainerregistry.NewRegistriesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &registriesClient{
		RegistriesClient: client,
	}
}
