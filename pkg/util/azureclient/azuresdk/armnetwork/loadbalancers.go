package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// LoadBalancersClient is a minimal interface for Azure LoadBalancersClient
type LoadBalancersClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string, options *armnetwork.LoadBalancersClientGetOptions) (result armnetwork.LoadBalancersClientGetResponse, err error)
	LoadBalancersClientAddons
}

type loadBalancersClient struct {
	*armnetwork.LoadBalancersClient
}

var _ LoadBalancersClient = &loadBalancersClient{}

// NewLoadBalancersClient creates a new LoadBalancersClient
func NewLoadBalancersClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (LoadBalancersClient, error) {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, &options)
	if err != nil {
		return nil, err
	}
	return &loadBalancersClient{LoadBalancersClient: clientFactory.NewLoadBalancersClient()}, nil
}

type LoadBalancerBackendAddressPoolsClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string, backendAddressPoolName string, options *armnetwork.LoadBalancerBackendAddressPoolsClientGetOptions) (result armnetwork.LoadBalancerBackendAddressPoolsClientGetResponse, err error)
}

type loadBalancerBackendAddressPoolsClient struct {
	*armnetwork.LoadBalancerBackendAddressPoolsClient
}

var _ LoadBalancerBackendAddressPoolsClient = &loadBalancerBackendAddressPoolsClient{}

// NewLoadBalancerBackendAddressPoolsClient creates a new NewLoadBalancerBackendAddressPoolsClient
func NewLoadBalancerBackendAddressPoolsClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (LoadBalancerBackendAddressPoolsClient, error) {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, &options)
	if err != nil {
		return nil, err
	}
	return &loadBalancerBackendAddressPoolsClient{LoadBalancerBackendAddressPoolsClient: clientFactory.NewLoadBalancerBackendAddressPoolsClient()}, nil
}
