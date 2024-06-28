package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// SecurityGroupsClient is a minimal interface for azure SecurityGroupsClient
type SecurityGroupsClient interface {
	Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityGroupsClientGetOptions) (armnetwork.SecurityGroupsClientGetResponse, error)
	SecurityGroupsClientAddons
}

type securityGroupsClient struct {
	*armnetwork.SecurityGroupsClient
}

// NewSecurityGroupsClient creates a new SecurityGroupsClient
func NewSecurityGroupsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (SecurityGroupsClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &securityGroupsClient{SecurityGroupsClient: clientFactory.NewSecurityGroupsClient()}, nil
}
