package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// PermissionsClient is a minimal interface for azure PermissionsClient
type PermissionsClient interface {
	PermissionsClientAddons
}

type permissionsClient struct {
	mgmtauthorization.PermissionsClient
}

var _ PermissionsClient = &permissionsClient{}

// NewPermissionsClient creates a new PermissionsClient
func NewPermissionsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) PermissionsClient {
	client := mgmtauthorization.NewPermissionsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &permissionsClient{
		PermissionsClient: client,
	}
}
