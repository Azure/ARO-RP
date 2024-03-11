package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
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
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &loadBalancersClient{
		LoadBalancersClient: client,
	}
}

type LoadBalancerBackendAddressPoolsClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string, backendAddressPoolName string) (result mgmtnetwork.BackendAddressPool, err error)
}

type loadBalancerBackendAddressPoolsClient struct {
	mgmtnetwork.LoadBalancerBackendAddressPoolsClient
}

var _ LoadBalancerBackendAddressPoolsClient = &loadBalancerBackendAddressPoolsClient{}

// NewLoadBalancerBackendAddressPoolsClient creates a new NewLoadBalancerBackendAddressPoolsClient
func NewLoadBalancerBackendAddressPoolsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) LoadBalancerBackendAddressPoolsClient {
	client := mgmtnetwork.NewLoadBalancerBackendAddressPoolsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &loadBalancerBackendAddressPoolsClient{
		LoadBalancerBackendAddressPoolsClient: client,
	}
}
