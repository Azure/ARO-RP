package armmsi

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type UserAssignedIdentitiesClient interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string, options *armmsi.UserAssignedIdentitiesClientGetOptions) (armmsi.UserAssignedIdentitiesClientGetResponse, error)
	GetClusterMSICredential() azcore.TokenCredential
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
