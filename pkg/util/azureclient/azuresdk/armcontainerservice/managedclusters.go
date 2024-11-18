package containerservice

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ManagedClustersClient is a minimal interface for azure ManagedClustersClient
type ManagedClustersClient interface {
	ManagedClustersAddons
}

type managedClustersClient struct {
	*armcontainerservice.ManagedClustersClient
}

var _ ManagedClustersClient = &managedClustersClient{}

// NewDefaultManagedClustersClient creates a new ManagedClustersClient with default options
func NewDefaultManagedClustersClient(environment *azureclient.AROEnvironment, subscriptionId string, credential azcore.TokenCredential) (ManagedClustersClient, error) {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}

	return NewManagedClustersClient(subscriptionId, credential, options)
}

// NewManagedClustersClient creates a new ManagedClustersClient
func NewManagedClustersClient(subscriptionId string, credential azcore.TokenCredential, options arm.ClientOptions) (ManagedClustersClient, error) {
	clientFactory, err := armcontainerservice.NewClientFactory(subscriptionId, credential, &options)
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewManagedClustersClient()

	return &managedClustersClient{
		ManagedClustersClient: client,
	}, nil
}

// Creates a new ManagedClustersClient with specified transport, useful for testing with fakes
// https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/resourcemanager/containerservice/armcontainerservice/fake_example_test.go
func NewManagedClustersClientWithTransport(environment *azureclient.AROEnvironment, subscriptionId string, tokenizer azcore.TokenCredential, transporter policy.Transporter) (ManagedClustersClient, error) {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud:     environment.Cloud,
			Transport: transporter,
		},
	}

	return NewManagedClustersClient(subscriptionId, tokenizer, options)
}
