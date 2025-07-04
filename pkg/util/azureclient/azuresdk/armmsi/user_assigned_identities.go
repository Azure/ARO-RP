package armmsi

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type UserAssignedIdentitiesClient interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string, options *armmsi.UserAssignedIdentitiesClientGetOptions) (armmsi.UserAssignedIdentitiesClientGetResponse, error)
	GetClusterMSICredential() azcore.TokenCredential
	CreateOrUpdate(ctx context.Context, resourceGroupName string, resourceName string, parameters armmsi.Identity, options *armmsi.UserAssignedIdentitiesClientCreateOrUpdateOptions) (armmsi.UserAssignedIdentitiesClientCreateOrUpdateResponse, error)
	Delete(ctx context.Context, resourceGroupName string, resourceName string, options *armmsi.UserAssignedIdentitiesClientDeleteOptions) (armmsi.UserAssignedIdentitiesClientDeleteResponse, error)
	NewListByResourceGroupPager(resourceGroupName string, options *armmsi.UserAssignedIdentitiesClientListByResourceGroupOptions) *runtime.Pager[armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse]
	Update(ctx context.Context, resourceGroupName string, resourceName string, parameters armmsi.IdentityUpdate, options *armmsi.UserAssignedIdentitiesClientUpdateOptions) (armmsi.UserAssignedIdentitiesClientUpdateResponse, error)
}

type ArmUserAssignedIdentitiesClient struct {
	*armmsi.UserAssignedIdentitiesClient
	cred azcore.TokenCredential
}

var _ UserAssignedIdentitiesClient = &ArmUserAssignedIdentitiesClient{}

// NewUserAssignedIdentitiesClient creates a new UserAssignedIdentitiesClient
func NewUserAssignedIdentitiesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (*ArmUserAssignedIdentitiesClient, error) {
	clientFactory, err := armmsi.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &ArmUserAssignedIdentitiesClient{
		UserAssignedIdentitiesClient: clientFactory.NewUserAssignedIdentitiesClient(),
		cred:                         credential,
	}, nil
}

func (c *ArmUserAssignedIdentitiesClient) GetClusterMSICredential() azcore.TokenCredential {
	return c.cred
}
