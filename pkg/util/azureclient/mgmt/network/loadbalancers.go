package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// LoadBalancersClient is a minimal interface for Azure LoadBalancersClient
type LoadBalancersClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string, expand string) (result mgmtnetwork.LoadBalancer, err error)
	LoadBalancersClientAddons
}

type loadBalancersClient struct {
	mgmtnetwork.LoadBalancersClient
}

var _ LoadBalancersClient = &loadBalancersClient{}

// NewLoadBalancersClient creates a new LoadBalancersClient
func NewLoadBalancersClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) LoadBalancersClient {
	client := mgmtnetwork.NewLoadBalancersClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &loadBalancersClient{
		LoadBalancersClient: client,
	}
}
