package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
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
func NewLoadBalancersClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (LoadBalancersClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &loadBalancersClient{LoadBalancersClient: clientFactory.NewLoadBalancersClient()}, nil
}

// LoadBalancerLoadBalancingRulesClient is a minimal interface for Azure LoadBalancerLoadBalancingRulesClient
type LoadBalancerLoadBalancingRulesClient interface {
	HealthAndWait(ctx context.Context, resourceGroupName string, loadBalancerName string, loadBalancingRuleName string, options *armnetwork.LoadBalancerLoadBalancingRulesClientBeginHealthOptions) (armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse, error)
}

type loadBalancerLoadBalancingRulesClient struct {
	*armnetwork.LoadBalancerLoadBalancingRulesClient
}

var _ LoadBalancerLoadBalancingRulesClient = &loadBalancerLoadBalancingRulesClient{}

// NewLoadBalancerLoadBalancingRulesClient creates a new LoadBalancerLoadBalancingRulesClient
func NewLoadBalancerLoadBalancingRulesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (LoadBalancerLoadBalancingRulesClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &loadBalancerLoadBalancingRulesClient{LoadBalancerLoadBalancingRulesClient: clientFactory.NewLoadBalancerLoadBalancingRulesClient()}, nil
}

type LoadBalancerBackendAddressPoolsClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string, backendAddressPoolName string, options *armnetwork.LoadBalancerBackendAddressPoolsClientGetOptions) (result armnetwork.LoadBalancerBackendAddressPoolsClientGetResponse, err error)
}

type loadBalancerBackendAddressPoolsClient struct {
	*armnetwork.LoadBalancerBackendAddressPoolsClient
}

var _ LoadBalancerBackendAddressPoolsClient = &loadBalancerBackendAddressPoolsClient{}

// NewLoadBalancerBackendAddressPoolsClient creates a new NewLoadBalancerBackendAddressPoolsClient
func NewLoadBalancerBackendAddressPoolsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (LoadBalancerBackendAddressPoolsClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &loadBalancerBackendAddressPoolsClient{LoadBalancerBackendAddressPoolsClient: clientFactory.NewLoadBalancerBackendAddressPoolsClient()}, nil
}
