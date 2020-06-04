package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2019-12-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
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
func NewRegistriesClient(subscriptionID string, authorizer autorest.Authorizer) RegistriesClient {
	client := mgmtcontainerregistry.NewRegistriesClient(subscriptionID)
	client.Authorizer = authorizer

	return &registriesClient{
		RegistriesClient: client,
	}
}
