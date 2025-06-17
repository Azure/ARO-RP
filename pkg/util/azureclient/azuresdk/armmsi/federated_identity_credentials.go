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

// FederatedIdentityCredentialsClient is a minimal interface for azure FederatedIdentityCredentialsClient
type FederatedIdentityCredentialsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, parameters armmsi.FederatedIdentityCredential, options *armmsi.FederatedIdentityCredentialsClientCreateOrUpdateOptions) (armmsi.FederatedIdentityCredentialsClientCreateOrUpdateResponse, error)
	Delete(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, options *armmsi.FederatedIdentityCredentialsClientDeleteOptions) (armmsi.FederatedIdentityCredentialsClientDeleteResponse, error)
	Get(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, options *armmsi.FederatedIdentityCredentialsClientGetOptions) (armmsi.FederatedIdentityCredentialsClientGetResponse, error)
	NewListPager(resourceGroupName string, resourceName string, options *armmsi.FederatedIdentityCredentialsClientListOptions) *runtime.Pager[armmsi.FederatedIdentityCredentialsClientListResponse]
	List(ctx context.Context, resourceGroupName string, resourceName string, options *armmsi.FederatedIdentityCredentialsClientListOptions) ([]*armmsi.FederatedIdentityCredential, error)
}

type ArmFederatedIdentityCredentialsClient struct {
	*armmsi.FederatedIdentityCredentialsClient
}

var _ FederatedIdentityCredentialsClient = &ArmFederatedIdentityCredentialsClient{}

// NewFederatedIdentityCredentialsClient creates a new FederatedIdentityCredentialsClient
func NewFederatedIdentityCredentialsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (*ArmFederatedIdentityCredentialsClient, error) {
	clientFactory, err := armmsi.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &ArmFederatedIdentityCredentialsClient{FederatedIdentityCredentialsClient: clientFactory.NewFederatedIdentityCredentialsClient()}, nil
}

func (c *ArmFederatedIdentityCredentialsClient) List(ctx context.Context, resourceGroupName string, resourceName string, options *armmsi.FederatedIdentityCredentialsClientListOptions) (result []*armmsi.FederatedIdentityCredential, err error) {
	pager := c.NewListPager(resourceGroupName, resourceName, options)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		result = append(result, page.Value...)
	}
	return result, nil
}
