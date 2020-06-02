package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
)

// RoleAssignmentsClient is a minimal interface for azure RoleAssignmentsClient
type RoleAssignmentsClient interface {
	RoleAssignmentsClientAddon
	Create(ctx context.Context, scope string, roleAssignmentName string, parameters mgmtauthorization.RoleAssignmentCreateParameters) (result mgmtauthorization.RoleAssignment, err error)
	DeleteByID(ctx context.Context, roleID string) (result authorization.RoleAssignment, err error)
}

type roleAssignmentsClient struct {
	mgmtauthorization.RoleAssignmentsClient
}

var _ RoleAssignmentsClient = &roleAssignmentsClient{}

// NewRoleAssignmentsClient creates a new RoleAssignmentsClient
func NewRoleAssignmentsClient(subscriptionID string, authorizer autorest.Authorizer) RoleAssignmentsClient {
	client := mgmtauthorization.NewRoleAssignmentsClient(subscriptionID)
	client.Authorizer = authorizer

	return &roleAssignmentsClient{
		RoleAssignmentsClient: client,
	}
}
