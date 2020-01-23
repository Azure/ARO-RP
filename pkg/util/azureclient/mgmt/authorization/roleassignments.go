package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/go-autorest/autorest"
)

// RoleAssignmentsClient is a minimal interface for azure RoleAssignmentsClient
type RoleAssignmentsClient interface {
	Create(ctx context.Context, scope string, roleAssignmentName string, parameters authorization.RoleAssignmentCreateParameters) (result authorization.RoleAssignment, err error)
}

type roleAssignmentsClient struct {
	authorization.RoleAssignmentsClient
}

var _ RoleAssignmentsClient = &roleAssignmentsClient{}

// NewRoleAssignmentsClient creates a new RoleAssignmentsClient
func NewRoleAssignmentsClient(subscriptionID string, authorizer autorest.Authorizer) RoleAssignmentsClient {
	client := authorization.NewRoleAssignmentsClient(subscriptionID)
	client.Authorizer = authorizer

	return &roleAssignmentsClient{
		RoleAssignmentsClient: client,
	}
}
