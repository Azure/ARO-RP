package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../../../vendor/github.com/golang/mock/mockgen -destination=../../../../util/mocks/azureclient/mgmt/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/$GOPACKAGE RoleAssignmentsClient
//go:generate go run ../../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../../util/mocks/azureclient/mgmt/$GOPACKAGE/$GOPACKAGE.go

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
