package armauthorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

type RoleDefinitionsClient interface {
	GetByID(ctx context.Context, roleID string, options *armauthorization.RoleDefinitionsClientGetByIDOptions) (armauthorization.RoleDefinitionsClientGetByIDResponse, error)
}

type roleDefinitionsClient struct {
	armauthorization.RoleDefinitionsClient
}

var _ RoleDefinitionsClient = &roleDefinitionsClient{}

func NewRoleDefinitionsClient(credential azcore.TokenCredential, options *arm.ClientOptions) (RoleDefinitionsClient, error) {
	return armauthorization.NewRoleDefinitionsClient(credential, options)
}
