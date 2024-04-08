package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// SecurityGroupsClient is a minimal interface for azure SecurityGroupsClient
type SecurityGroupsClient interface {
	Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, expand string) (result mgmtnetwork.SecurityGroup, err error)
	SecurityGroupsClientAddons
}

type securityGroupsClient struct {
	mgmtnetwork.SecurityGroupsClient
}

var _ SecurityGroupsClient = &securityGroupsClient{}

// NewSecurityGroupsClient creates a new SecurityGroupsClient
func NewSecurityGroupsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) SecurityGroupsClient {
	client := mgmtnetwork.NewSecurityGroupsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &securityGroupsClient{
		SecurityGroupsClient: client,
	}
}
