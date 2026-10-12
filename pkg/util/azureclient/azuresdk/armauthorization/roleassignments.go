package armauthorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

// RoleAssignmentsClient is a minimal interface for Azure RoleAssignmentsClient
type RoleAssignmentsClient interface {
	Create(ctx context.Context, scope string, roleAssignmentName string, parameters armauthorization.RoleAssignmentCreateParameters, options *armauthorization.RoleAssignmentsClientCreateOptions) (armauthorization.RoleAssignmentsClientCreateResponse, error)
	RoleAssignmentsClientAddons
}

type roleAssignmentsClient struct {
	*armauthorization.RoleAssignmentsClient
}

var _ RoleAssignmentsClient = &roleAssignmentsClient{}

// NewRoleAssignmentsClient creates a new RoleAssignmentsClient
func NewRoleAssignmentsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (RoleAssignmentsClient, error) {
	client, err := armauthorization.NewRoleAssignmentsClient(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &roleAssignmentsClient{RoleAssignmentsClient: client}, nil
}
