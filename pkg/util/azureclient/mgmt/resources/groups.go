package resources

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

// GroupsClient is a minimal interface for azure GroupsClient
type GroupsClient interface {
	Get(ctx context.Context, resourceGroupName string) (result resources.Group, err error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, parameters resources.Group) (result resources.Group, err error)
	GroupsClientAddons
}

type groupsClient struct {
	resources.GroupsClient
}

var _ GroupsClient = &groupsClient{}

// NewGroupsClient creates a new ResourcesClient
func NewGroupsClient(subscriptionID string, authorizer autorest.Authorizer) GroupsClient {
	client := resources.NewGroupsClient(subscriptionID)
	client.Authorizer = authorizer
	client.PollingDuration = time.Hour

	return &groupsClient{
		GroupsClient: client,
	}
}
