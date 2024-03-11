package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// RoleDefinitionsClient is a minimal interface for azure RoleDefinitionsClient
type RoleDefinitionsClient interface {
	Delete(ctx context.Context, scope string, roleDefinitionID string) (result mgmtauthorization.RoleDefinition, err error)
	RoleDefinitionsClientAddons
}

type roleDefinitionsClient struct {
	mgmtauthorization.RoleDefinitionsClient
}

var _ RoleDefinitionsClient = &roleDefinitionsClient{}

// NewRoleDefinitionsClient creates a new RoleDefinitionsClient
func NewRoleDefinitionsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) RoleDefinitionsClient {
	client := mgmtauthorization.NewRoleDefinitionsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &roleDefinitionsClient{
		RoleDefinitionsClient: client,
	}
}
