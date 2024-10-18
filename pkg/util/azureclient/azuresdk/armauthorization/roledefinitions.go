package armauthorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

type RoleDefinitionsClient interface {
	GetByID(ctx context.Context, roleID string, options *armauthorization.RoleDefinitionsClientGetByIDOptions) (armauthorization.RoleDefinitionsClientGetByIDResponse, error)
	Get(ctx context.Context, scope string, roleDefinitionID string, options *armauthorization.RoleDefinitionsClientGetOptions) (armauthorization.RoleDefinitionsClientGetResponse, error)
}

type ArmRoleDefinitionsClient struct {
	*armauthorization.RoleDefinitionsClient
	subscriptionID string
}

var _ RoleDefinitionsClient = &ArmRoleDefinitionsClient{}

func NewArmRoleDefinitionsClient(credential azcore.TokenCredential, subscriptionID string, options *arm.ClientOptions) (*ArmRoleDefinitionsClient, error) {
	client, err := armauthorization.NewRoleDefinitionsClient(credential, options)
	return &ArmRoleDefinitionsClient{
		RoleDefinitionsClient: client,
		subscriptionID:        subscriptionID,
	}, err
}

func (client ArmRoleDefinitionsClient) GetByID(ctx context.Context, roleDefinitionID string, options *armauthorization.RoleDefinitionsClientGetByIDOptions) (armauthorization.RoleDefinitionsClientGetByIDResponse, error) {
	roleID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s", client.subscriptionID, roleDefinitionID)
	return client.RoleDefinitionsClient.GetByID(ctx, roleID, options)
}
