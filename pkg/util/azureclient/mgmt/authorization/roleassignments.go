package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// RoleAssignmentsClient is a minimal interface for azure RoleAssignmentsClient
type RoleAssignmentsClient interface {
	Create(ctx context.Context, scope string, roleAssignmentName string, parameters mgmtauthorization.RoleAssignmentCreateParameters) (result mgmtauthorization.RoleAssignment, err error)
	Delete(ctx context.Context, scope string, roleAssignmentName string) (result mgmtauthorization.RoleAssignment, err error)
	RoleAssignmentsClientAddons
}

type roleAssignmentsClient struct {
	mgmtauthorization.RoleAssignmentsClient
}

var _ RoleAssignmentsClient = &roleAssignmentsClient{}

// NewRoleAssignmentsClient creates a new RoleAssignmentsClient
func NewRoleAssignmentsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) RoleAssignmentsClient {
	client := mgmtauthorization.NewRoleAssignmentsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &roleAssignmentsClient{
		RoleAssignmentsClient: client,
	}
}
